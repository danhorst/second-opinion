//go:build integration

package claude_test

import (
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/claude"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

// The claude adapter passes the shared conformance suite against the real
// binary. Requires `claude` on PATH and an authenticated session; run via
// `make test-integration`.
func TestClaudeConformance(t *testing.T) {
	providertest.Conform(t, func(t *testing.T) provider.Provider {
		return claude.New()
	})
}

// Provenance names the model that actually ran, from the harness's own
// usage report.
func TestClaudeProvenance(t *testing.T) {
	res, err := claude.New().Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if !strings.HasPrefix(res.Provenance.Model, "claude-") || res.Provenance.Model == "claude-default" {
		t.Errorf("provenance did not capture the model that ran: %+v", res.Provenance)
	}
}
