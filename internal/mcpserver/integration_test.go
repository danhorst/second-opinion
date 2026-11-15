//go:build integration

package mcpserver

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

// These calls exercise the actual SDK and installed reviewer. They use only
// synthetic material, and supplement rather than replace cold conformance.
func TestRealMCPReviews(t *testing.T) {
	for _, p := range []struct{ name, binary string }{{"codex", "codex"}, {"antigravity", "agy"}, {"claude", "claude"}} {
		t.Run(p.name, func(t *testing.T) {
			if _, err := exec.LookPath(p.binary); err != nil {
				t.Skipf("real provider unavailable: %s", p.binary)
			}
			cs := sdkSession(t, review.Executor{}, "2026-07-28")
			prompt := "Repeat the exact marker found in the supplied material. Do not add any other text."
			material := "MCP_POSITIVE_CONTROL_7b3de912"
			result := callTool(t, cs, "review", map[string]any{"provider": p.name, "prompt": prompt, "material": material, "timeout_ms": 120000})
			value := mirrored(t, result)
			if result.IsError {
				t.Fatalf("real MCP review failed: %+v", value)
			}
			if !strings.Contains(value["raw"].(string), material) {
				t.Fatal("positive control marker missing")
			}
			provenance := value["provenance"].(map[string]any)
			if provenance["provider"] != p.name || provenance["model"] == "" || provenance["prompt_hash"] != provider.HashPrompt(prompt) {
				t.Fatalf("invalid provenance %+v", provenance)
			}
			t.Logf("real MCP positive control passed: provider=%s model=%s", p.name, provenance["model"])
		})
	}
}
