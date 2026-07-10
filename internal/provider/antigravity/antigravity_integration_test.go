//go:build integration

package antigravity_test

import (
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/antigravity"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

// The antigravity adapter passes the shared conformance suite against the
// real binary. Requires `agy` on PATH and an authenticated session; run via
// `make test-integration`.
func TestAntigravityConformance(t *testing.T) {
	providertest.Conform(t, func(t *testing.T) provider.Provider {
		return antigravity.New()
	})
}

// An unforced run carries the documented default marker; the review itself
// must succeed.
func TestAntigravityProvenanceUnforced(t *testing.T) {
	res, err := antigravity.New().Review(t.Context(), provider.Request{
		Prompt:   "Reply with the single word OK and nothing else.",
		Material: "no material",
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	t.Logf("provider=%s model=%s findings=%q", res.Provenance.Provider, res.Provenance.Model, res.Findings)
	if res.Provenance.Model != "antigravity-default" {
		t.Errorf("unforced provenance must carry the default marker, got %q", res.Provenance.Model)
	}
}
