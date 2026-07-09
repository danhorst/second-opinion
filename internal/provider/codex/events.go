package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
)

// parseEvents extracts the model that ran and the final agent message from
// codex's --json event stream. Parsing is defensive: unknown event shapes
// are skipped, and both the protocol shape ({"msg":{"type":"agent_message",
// "message":...}}) and the item shape ({"item":{"item_type":"agent_message",
// "text":...}}) are recognized. The integration test pins the shapes the
// installed binary actually emits.
func parseEvents(stream []byte) (model, findings string) {
	sc := bufio.NewScanner(bytes.NewReader(stream))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		if model == "" {
			model = findModel(event)
		}
		if text, ok := agentMessage(event); ok {
			findings = text // last agent message wins
		}
	}
	return model, findings
}

// errorEventMessage extracts the message from the first error event in the
// stream. Codex reports failures (usage limits, auth problems) as JSONL
// {"type":"error","message":...} events on stdout, not on stderr.
func errorEventMessage(stream []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(stream))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		if typ, _ := event["type"].(string); typ == "error" {
			if msg, ok := event["message"].(string); ok && msg != "" {
				return msg
			}
		}
	}
	return ""
}

// findModel walks an event for the first string-valued "model" key.
func findModel(v any) string {
	switch node := v.(type) {
	case map[string]any:
		if m, ok := node["model"].(string); ok && m != "" {
			return m
		}
		for _, child := range node {
			if m := findModel(child); m != "" {
				return m
			}
		}
	case []any:
		for _, child := range node {
			if m := findModel(child); m != "" {
				return m
			}
		}
	}
	return ""
}

// agentMessage walks an event for an agent-message object and returns its
// text.
func agentMessage(v any) (string, bool) {
	switch node := v.(type) {
	case map[string]any:
		typ, _ := node["type"].(string)
		itemType, _ := node["item_type"].(string)
		if typ == "agent_message" || itemType == "agent_message" {
			if s, ok := node["message"].(string); ok {
				return s, true
			}
			if s, ok := node["text"].(string); ok {
				return s, true
			}
		}
		for _, child := range node {
			if s, ok := agentMessage(child); ok {
				return s, ok
			}
		}
	case []any:
		for _, child := range node {
			if s, ok := agentMessage(child); ok {
				return s, ok
			}
		}
	}
	return "", false
}
