package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/provider/providertest"
)

func loopback(name, model string) (provider.Provider, error) {
	return providertest.NewLoopback(), nil
}

func noEnv(string) string { return "" }

func runCLI(t *testing.T, args []string, getenv func(string) string,
	newProvider func(string, string) (provider.Provider, error)) (code int, stdout, stderr string) {
	t.Helper()
	var out, errBuf strings.Builder
	code = run(context.Background(), args, getenv, &out, &errBuf, newProvider)
	return code, out.String(), errBuf.String()
}

func TestNoProviderRefusesWithList(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	os.WriteFile(f, []byte("content"), 0o644)

	code, _, stderr := runCLI(t, []string{f}, noEnv, loopback)
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	for _, name := range []string{"antigravity", "claude", "codex"} {
		if !strings.Contains(stderr, name) {
			t.Errorf("refusal must list %s: %q", name, stderr)
		}
	}
}

func TestEnvProvidesDefault(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	os.WriteFile(f, []byte("content"), 0o644)
	env := func(k string) string {
		if k == "SECOND_OPINION_PROVIDER" {
			return "loopback"
		}
		return ""
	}

	code, _, stderr := runCLI(t, []string{f}, env, loopback)
	if code != 0 {
		t.Errorf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "reviewed-by: provider=loopback") {
		t.Errorf("provenance line missing: %q", stderr)
	}
}

func TestFindingsOnStdoutProvenanceOnStderr(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "doc.md")
	os.WriteFile(f, []byte("content"), 0o644)

	code, stdout, stderr := runCLI(t, []string{"--provider", "loopback", f}, noEnv, loopback)
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if strings.Contains(stdout, "reviewed-by:") {
		t.Errorf("provenance leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stderr, "model=loopback-1") {
		t.Errorf("provenance missing from stderr: %q", stderr)
	}
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{},                                 // no target
		{"--diff", "--provider", "x", "a"}, // diff mode and paths are mutually exclusive
		{"--unknown-flag"},                 // unknown flag
		{"--model"},                        // missing value
	}
	for _, args := range cases {
		if code, _, _ := runCLI(t, args, noEnv, loopback); code != 2 {
			t.Errorf("args %v: exit = %d, want 2", args, code)
		}
	}
}

func TestEmptyMaterialExitsOne(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "empty.md")
	os.WriteFile(f, nil, 0o644)

	code, _, stderr := runCLI(t, []string{"--provider", "loopback", f}, noEnv, loopback)
	if code != 1 {
		t.Errorf("exit = %d, want 1 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "nothing to review") {
		t.Errorf("stderr must say nothing to review: %q", stderr)
	}
}

func TestPromptFileOverride(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	promptFile := filepath.Join(dir, "prompt.txt")
	os.WriteFile(doc, []byte("content"), 0o644)
	os.WriteFile(promptFile, []byte("custom prompt"), 0o644)

	var captured provider.Request
	capture := func(name, model string) (provider.Provider, error) {
		return captureProvider{&captured}, nil
	}
	code, _, stderr := runCLI(t, []string{"--provider", "x", "--prompt-file", promptFile, doc}, noEnv, capture)
	if code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, stderr)
	}
	if captured.Prompt != "custom prompt" {
		t.Errorf("override must replace the prompt wholesale, got %q", captured.Prompt)
	}
}

func TestParseArgsDiffOptionalBase(t *testing.T) {
	cfg, err := parseArgs([]string{"--diff"})
	if err != nil || !cfg.diffMode || cfg.diffBase != "HEAD" {
		t.Errorf("bare --diff: %+v, %v", cfg, err)
	}
	cfg, err = parseArgs([]string{"--diff", "main"})
	if err != nil || cfg.diffBase != "main" {
		t.Errorf("--diff main: %+v, %v", cfg, err)
	}
	if _, err := parseArgs([]string{"--diff", "a.txt", "b.txt"}); err == nil {
		t.Error("--diff BASE with extra paths must be mutually exclusive")
	}
}

type captureProvider struct{ req *provider.Request }

func (c captureProvider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	*c.req = req
	return &provider.Result{Provenance: provider.Provenance{
		Provider: "capture", Model: "capture-1", PromptHash: provider.HashPrompt(req.Prompt),
	}}, nil
}
