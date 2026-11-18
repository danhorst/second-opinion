package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWorstCaseRawEncodingAndOversizedProvenance(t *testing.T) {
	// NUL expands to six JSON bytes and expands again inside mirrored text.
	raw := strings.Repeat("\x00", MaxRawBytes)
	output := ReviewOutput{"unverified", 2, "review", raw, Provenance{"codex", "model", strings.Repeat("a", 64), strings.Repeat("b", 64)}, Receipt{strings.Repeat("b", 64), 1, 120000}}
	result := toolResult(output, false)
	if result.IsError {
		t.Fatal("maximum valid raw findings rejected")
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) >= MaxResponseBytes {
		t.Fatalf("encoded size=%d err=%v", len(encoded), err)
	}
	if mirrored(t, result)["raw"] != raw {
		t.Fatal("raw findings changed")
	}
	output.Provenance.Model = strings.Repeat("x", MaxResponseBytes)
	failureCode(t, toolResult(output, false), "output_too_large")
}
