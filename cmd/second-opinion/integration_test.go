//go:build integration

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/review"
)

// One real review end-to-end through the CLI wiring. Requires the claude
// binary and an authenticated session; run via `make test-integration`.
func TestCLISmokeRealProvider(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "design.md")
	os.WriteFile(doc, []byte("# Design\nThe cache is invalidated before the write completes, and readers assume it is always fresh.\n"), 0o644)

	var out, errBuf strings.Builder
	code := run(context.Background(), []string{"--provider", "claude", doc},
		func(string) string { return "" }, &out, &errBuf, review.NewProvider)
	if code != 0 {
		t.Fatalf("exit = %d\nstderr: %s", code, errBuf.String())
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Error("expected findings on stdout")
	}
	if !strings.Contains(errBuf.String(), "reviewed-by: provider=claude model=claude-") {
		t.Errorf("provenance line missing or untruthful: %q", errBuf.String())
	}
	t.Logf("stderr: %s", errBuf.String())
}
