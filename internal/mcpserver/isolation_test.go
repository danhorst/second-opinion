package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

func TestMCPIsolationPolicy(t *testing.T) {
	calls := 0
	e := review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
		return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) {
			calls++
			return goodResult(r), nil
		}), nil
	}}
	cs := sdkSession(t, e, "2025-11-25")
	r := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "m", "require_isolation": "local-material-only"})
	failureCode(t, r, "isolation_unsupported")
	if mirrored(t, r)["error"].(map[string]any)["execution"] != "not_started" || calls != 0 {
		t.Fatalf("execution started: calls=%d", calls)
	}
	for _, value := range []any{nil, "", "unverified", "future", true, 1, []string{}} {
		failureCode(t, callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "m", "require_isolation": value}), "invalid_input")
	}
	if calls != 0 {
		t.Fatal("invalid request invoked reviewer")
	}
	r = callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "m"})
	if r.IsError || mirrored(t, r)["isolation"] != "unverified" || calls != 1 {
		t.Fatal("default behavior changed")
	}
	discovery := mirrored(t, callTool(t, cs, "list_providers", map[string]any{}))["provider_isolation"].(map[string]any)
	if discovery["openai-compatible"] != "local-material-only" || discovery["claude"] != "unverified" {
		t.Fatalf("discovery=%v", discovery)
	}
}

// Real local HTTP transport through MCP and executor, not remote conformance.
func TestMCPMaterialOnlyHTTPReview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"model":"fixture","choices":[{"message":{"content":"NONE"}}]}`))
	}))
	defer server.Close()
	t.Setenv("SECOND_OPINION_API_BASE_URL", server.URL)
	t.Setenv("SECOND_OPINION_API_KEY", "fixture")
	t.Setenv("SECOND_OPINION_API_MODEL", "fixture")
	cs := sdkSession(t, review.Executor{}, "2025-11-25")
	r := callTool(t, cs, "review", map[string]any{"provider": "openai-compatible", "material": "m", "require_isolation": "local-material-only"})
	v := mirrored(t, r)
	if r.IsError || v["isolation"] != "local-material-only" || v["raw"] != "NONE" {
		t.Fatalf("result=%v", v)
	}
}

func TestMCPProviderIsolationErrorDoesNotClaimNotStarted(t *testing.T) {
	calls := 0
	e := review.Executor{NewProvider: func(string, string, provider.CaptureLimits) (provider.Provider, error) {
		return testReviewer(func(_ context.Context, r provider.Request) (*provider.Result, error) {
			calls++
			return nil, provider.ErrIsolationUnsupported
		}), nil
	}}
	cs := sdkSession(t, e, "2025-11-25")
	r := callTool(t, cs, "review", map[string]any{"provider": "codex", "material": "m"})
	failureCode(t, r, "provider_failed")
	if calls != 1 || mirrored(t, r)["error"].(map[string]any)["execution"] != "unknown" {
		t.Fatal("false not-started receipt")
	}
}
