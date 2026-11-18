package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

func TestCLIIsolationPolicy(t *testing.T) {
	for _, args := range [][]string{{"--require-isolation"}, {"--require-isolation", "unknown", "doc"}, {"--require-isolation", "unverified", "doc"}} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	calls := 0
	factory := func(string, string) (provider.Provider, error) {
		return reviewFunc(func(context.Context, provider.Request) (*provider.Result, error) {
			calls++
			return &provider.Result{}, nil
		}), nil
	}
	code, out, errout := runCLI(t, []string{"--provider", "test", "--require-isolation", "local-material-only", controlTarget(t)}, noEnv, factory)
	if code != 1 || out != "" || !strings.Contains(errout, "required isolation unsupported") || calls != 0 {
		t.Fatalf("code=%d out=%q err=%q calls=%d", code, out, errout, calls)
	}
	code, out, errout = runCLI(t, []string{"--dry-run", "--require-isolation", "local-material-only", controlTarget(t)}, noEnv, func(string, string) (provider.Provider, error) {
		t.Fatal("preview constructed provider")
		return nil, nil
	})
	var preview map[string]any
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatal(err)
	}
	if code != 0 || preview["require_isolation"] != "local-material-only" || preview["isolation"] != nil {
		t.Fatalf("preview=%v code=%d err=%s", preview, code, errout)
	}
	code, out, errout = runCLI(t, []string{"--list-providers", "--json"}, noEnv, factory)
	var discovery struct {
		ProviderIsolation map[string]string `json:"provider_isolation"`
	}
	if err := json.Unmarshal([]byte(out), &discovery); err != nil {
		t.Fatal(err)
	}
	if code != 0 || discovery.ProviderIsolation["openai-compatible"] != "local-material-only" || discovery.ProviderIsolation["codex"] != "unverified" || calls != 0 {
		t.Fatalf("discovery=%+v code=%d err=%s", discovery, code, errout)
	}
}

// Actual HTTP adapter through CLI wiring; this is local transport evidence only.
func TestCLIMaterialOnlyHTTPReview(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"model":"fixture","choices":[{"message":{"content":"NONE"}}]}`))
	}))
	defer server.Close()
	t.Setenv("SECOND_OPINION_API_BASE_URL", server.URL)
	t.Setenv("SECOND_OPINION_API_KEY", "fixture")
	t.Setenv("SECOND_OPINION_API_MODEL", "fixture")
	code, out, errout := runCLI(t, []string{"--provider", "openai-compatible", "--require-isolation", "local-material-only", "--json", controlTarget(t)}, noEnv, review.NewProvider)
	var result reviewOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decode: %v, stderr=%s", err, errout)
	}
	if code != 0 || result.Isolation != provider.IsolationLocalMaterialOnly || result.Raw != "NONE" || !strings.Contains(errout, "isolation=local-material-only") {
		t.Fatalf("code=%d result=%+v err=%s", code, result, errout)
	}
}
