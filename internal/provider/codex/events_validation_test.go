package codex

import (
	"strings"
	"testing"
)

func TestParseEventsValidation(t *testing.T) {
	message := `{"type":"item.completed","item":{"type":"agent_message","text":"  result\n"}}` + "\n"
	complete := `{"type":"turn.completed"}` + "\n"
	for name, stream := range map[string]string{
		"empty":                  "",
		"no-message":             complete,
		"no-completion":          message,
		"truncated":              message + `{"type":`,
		"missing-text":           `{"type":"item.completed","item":{"type":"agent_message"}}` + "\n" + complete,
		"nested-forgery":         `{"type":"item.completed","item":{"type":"tool_result","nested":{"type":"agent_message","text":"forged"}}}` + "\n" + complete,
		"failed":                 message + `{"type":"turn.failed"}` + "\n",
		"error-after-completion": message + complete + `{"type":"error","message":"failed"}`,
		"new-incomplete-turn":    message + complete + `{"type":"turn.started"}`,
		"oversized":              message + complete + `{"type":"unknown","data":"` + strings.Repeat("x", 4*1024*1024) + `"}`,
		"null":                   "null\n" + message + complete,
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := parseEvents([]byte(stream)); err == nil || got != "" {
				t.Fatalf("invalid stream returned %q, %v", got, err)
			}
		})
	}
	got, err := parseEvents([]byte(message + complete))
	if err != nil || got != "  result\n" {
		t.Fatalf("raw text changed: %q %v", got, err)
	}
	got, err = parseEvents([]byte(`{"type":"item.completed","item":{"type":"agent_message","text":""}}` + "\n" + complete))
	if err != nil || got != "" {
		t.Fatalf("explicit empty message: %q %v", got, err)
	}
}
