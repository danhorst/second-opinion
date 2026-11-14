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

// An unforced run carries the documented default marker; the review itself
// must succeed. Codex's --json event stream carries no model field to
// extract (checked against the real binary), so this marker is the honest
// answer, not a parsing gap.
func TestCodexProvenanceUnforced(t *testing.T) {
	res, err := codex.New().Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model != "codex-default" {
		t.Errorf("unforced provenance must carry the default marker, got %q", res.Provenance.Model)
	}
}

// A forced run's provenance names the requested model.
func TestCodexProvenanceForced(t *testing.T) {
	res, err := codex.New(codex.WithModel("gpt-5.4-mini")).Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model != "gpt-5.4-mini" {
		t.Errorf("forced provenance must name the requested model, got %q", res.Provenance.Model)
	}
}
