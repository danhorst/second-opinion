//go:build integration

package codex_test

import (
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/codex"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

// The codex adapter passes the shared conformance suite against the real
// binary. Requires `codex` on PATH and active auth; run via
// `make test-integration`.
func TestCodexConformance(t *testing.T) {
	providertest.Conform(t, func(t *testing.T) provider.Provider {
		return codex.New()
	})
}

// Verified against codex-cli 0.149.0-alpha.4.3 on 2026-08-25.
// Current unforced JSONL events omit model identity, so codex-default is the
// documented marker for that case.
func TestCodexProvenance(t *testing.T) {
	res, err := codex.New().Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model == "" {
		t.Errorf("provenance did not report a model or documented marker: %+v", res.Provenance)
	}
}
