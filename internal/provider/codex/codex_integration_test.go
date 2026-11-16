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

// Real runs preserve reported model identity, or the explicit default marker
// when the installed Codex event schema does not expose it.
func TestCodexProvenanceUnforced(t *testing.T) {
	res, err := codex.New().Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model == "" {
		t.Error("unforced provenance must contain reported model or default marker")
	}
}

// A forced run preserves reported model metadata; requested model is fallback.
func TestCodexProvenanceForced(t *testing.T) {
	res, err := codex.New(codex.WithModel("gpt-5.4-mini")).Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model == "" {
		t.Error("forced provenance is missing")
	}
}
