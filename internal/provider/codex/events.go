package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type messageEvent struct {
	Type     string  `json:"type"`
	Model    string  `json:"model"`
	ItemType string  `json:"item_type"`
	Message  *string `json:"message"`
	Text     *string `json:"text"`
}

type streamEvent struct {
	Type    string `json:"type"`
	Model   string `json:"model"`
	Session *struct {
		Model string `json:"model"`
	} `json:"session"`
	Item *messageEvent `json:"item"`
	Msg  *messageEvent `json:"msg"`
}

// parseEvents validates transport completion separately from findings content.
// Unknown typed events are tolerated, but arbitrary nested messages cannot
// become findings. Explicit empty messages are valid; missing messages are not.
func parseEvents(stream []byte) (model, findings string, err error) {
	sc := bufio.NewScanner(bytes.NewReader(stream))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	seen, completed := false, false
	lineNumber := 0
	for sc.Scan() {
		lineNumber++
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return "", "", fmt.Errorf("codex: malformed JSONL event at line %d", lineNumber)
		}
		typ := event.Type
		if typ == "" && event.Msg != nil {
			typ = event.Msg.Type
		}
		if typ == "" {
			return "", "", fmt.Errorf("codex: event at line %d has no type", lineNumber)
		}
		// Only session metadata establishes model identity; arbitrary nested
		// tool payloads and reviewer text must not forge provenance.
		switch typ {
		case "session.created", "session_configured", "thread.started":
			if event.Model != "" {
				model = event.Model
			}
			if event.Session != nil && event.Session.Model != "" {
				model = event.Session.Model
			}
			if event.Msg != nil && event.Msg.Model != "" {
				model = event.Msg.Model
			}
		}
		var message *messageEvent
		switch typ {
		case "error", "turn.failed", "turn_aborted":
			return "", "", fmt.Errorf("codex: event stream reports %s", typ)
		case "turn.started", "task_started":
			seen, completed, findings = false, false, ""
		case "turn.completed", "task_complete":
			if !seen {
				return "", "", fmt.Errorf("codex: completed turn has no agent message")
			}
			completed = true
		case "item.completed":
			if event.Item != nil && (event.Item.Type == "agent_message" || event.Item.ItemType == "agent_message") {
				message = event.Item
			}
		case "agent_message":
			// Legacy protocol only: the message must be directly under msg.
			if event.Msg != nil && event.Msg.Type == "agent_message" {
				message = event.Msg
			}
		}
		if message != nil {
			switch {
			case message.Text != nil:
				findings = *message.Text
			case message.Message != nil:
				findings = *message.Message
			default:
				return "", "", fmt.Errorf("codex: agent message has no text")
			}
			seen, completed = true, false
		}
	}
	if err := sc.Err(); err != nil {
		return "", "", fmt.Errorf("codex: reading JSONL events: %w", err)
	}
	if !seen || !completed {
		return "", "", fmt.Errorf("codex: incomplete event stream: missing agent message or terminal completion")
	}
	return model, findings, nil
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
		if typ, _ := event["type"].(string); typ == "error" || typ == "turn.failed" || typ == "turn_aborted" {
			if msg, ok := event["message"].(string); ok && msg != "" {
				return msg
			}
			if detail, ok := event["error"].(map[string]any); ok {
				if msg, ok := detail["message"].(string); ok {
					return msg
				}
			}
		}
	}
	return ""
}
