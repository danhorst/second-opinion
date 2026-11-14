package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/provider"
)

// This seam tests front-end orchestration, not any provider's transport.
type reviewFunc func(context.Context, provider.Request) (*provider.Result, error)

func (f reviewFunc) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	return f(ctx, req)
}

func controlTarget(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/doc.md"
	if err := os.WriteFile(path, []byte("material\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestControlArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--provider", "--diff"}, {"--model", "", "doc"},
		{"--prompt-file", "--help", "doc"}, {"--diff", ""},
		{"--timeout", "0", "doc"}, {"--timeout", "-1s", "doc"},
		{"--timeout", "tomorrow", "doc"}, {"--timeout"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("accepted invalid arguments: %q", args)
		}
	}
	cfg, err := parseArgs([]string{"--json", "--timeout", "2m", "--", "--help", "--diff"})
	if err != nil || !cfg.jsonOutput || cfg.timeout != 2*time.Minute || cfg.help || cfg.diffMode || strings.Join(cfg.paths, ",") != "--help,--diff" {
		t.Fatalf("literal paths: %+v, %v", cfg, err)
	}
}

func TestPreviewMatchesReviewRequest(t *testing.T) {
	doc := controlTarget(t)
	prompt := t.TempDir() + "/prompt.txt"
	if err := os.WriteFile(prompt, []byte("custom prompt\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	noProvider := func(string, string) (provider.Provider, error) {
		t.Fatal("preview constructed a provider")
		return nil, nil
	}
	code, stdout, stderr := runCLI(t, []string{"--dry-run", "--provider", "ignored", "--prompt-file", prompt, doc}, noEnv, noProvider)
	if code != 0 || stderr != "" {
		t.Fatalf("preview: %d, %q", code, stderr)
	}
	var preview requestOutput
	if err := json.Unmarshal([]byte(stdout), &preview); err != nil {
		t.Fatal(err)
	}
	var actual provider.Request
	capture := func(string, string) (provider.Provider, error) { return captureProvider{&actual}, nil }
	code, _, stderr = runCLI(t, []string{"--provider", "x", "--prompt-file", prompt, doc}, noEnv, capture)
	if code != 0 || preview.Kind != "request" || preview.SchemaVersion != 1 || preview.Prompt != actual.Prompt || preview.Material != actual.Material || preview.PromptHash != provider.HashPrompt(actual.Prompt) {
		t.Fatalf("preview mismatch: %+v, actual %+v, code %d, %s", preview, actual, code, stderr)
	}
	code, _, stderr = runCLI(t, []string{"--dry-run", doc}, noEnv, noProvider)
	if code != 0 || stderr != "" {
		t.Fatalf("provider-free preview: %d %s", code, stderr)
	}
}

func TestJSONPreservesRawAndProvenance(t *testing.T) {
	for _, raw := range []string{"", " \r\n>>[major] a.go:2\nquote: \"<&>\" 日本語\n\t"} {
		t.Run(raw, func(t *testing.T) {
			p := func(string, string) (provider.Provider, error) {
				return reviewFunc(func(_ context.Context, req provider.Request) (*provider.Result, error) {
					return &provider.Result{Findings: raw, Provenance: provider.Provenance{
						Provider: "test", Model: "unknown-default", PromptHash: provider.HashPrompt(req.Prompt),
					}}, nil
				}), nil
			}
			code, stdout, stderr := runCLI(t, []string{"--json", "--provider", "test", controlTarget(t)}, noEnv, p)
			if code != 0 || !strings.Contains(stderr, "reviewed-by:") {
				t.Fatalf("run: %d %s", code, stderr)
			}
			var result reviewOutput
			decoder := json.NewDecoder(strings.NewReader(stdout))
			if err := decoder.Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.Kind != "review" || result.SchemaVersion != 1 || result.Raw != raw || result.Provenance.Provider != "test" || result.Provenance.Model != "unknown-default" || len(result.Provenance.PromptHash) != 64 {
				t.Fatalf("wrong JSON: %+v", result)
			}
			if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
				t.Fatalf("extra stdout content: %v", err)
			}
		})
	}
}

func TestReviewDeadline(t *testing.T) {
	p := func(string, string) (provider.Provider, error) {
		return reviewFunc(func(ctx context.Context, _ provider.Request) (*provider.Result, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("provider received no deadline")
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}), nil
	}
	code, stdout, stderr := runCLI(t, []string{"--json", "--provider", "test", "--timeout", "1ns", controlTarget(t)}, noEnv, p)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "deadline exceeded") {
		t.Fatalf("deadline: %d %q %q", code, stdout, stderr)
	}
}

func TestJSONFailureHasNoSuccessObject(t *testing.T) {
	p := func(string, string) (provider.Provider, error) {
		return reviewFunc(func(ctx context.Context, _ provider.Request) (*provider.Result, error) {
			if _, ok := ctx.Deadline(); ok {
				t.Fatal("unexpected default deadline")
			}
			return nil, provider.ErrUnavailable
		}), nil
	}
	code, stdout, stderr := runCLI(t, []string{"--json", "--provider", "test", controlTarget(t)}, noEnv, p)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "unavailable") {
		t.Fatalf("failure: %d %q %q", code, stdout, stderr)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken output") }

func TestOutputFailure(t *testing.T) {
	for _, flag := range []string{"--json", "--dry-run", ""} {
		args := []string{"--provider", "test", controlTarget(t)}
		if flag != "" {
			args = append(args, flag)
		}
		var stderr strings.Builder
		code := run(context.Background(), args, noEnv, failingWriter{}, &stderr, loopback)
		if code != 1 || !strings.Contains(stderr.String(), "writing output") || strings.Contains(stderr.String(), "did not run") {
			t.Fatalf("%s: %d %s", flag, code, stderr.String())
		}
	}
}

func TestJSONRejectsInvalidUTF8WithoutReplacement(t *testing.T) {
	doc := controlTarget(t)
	if err := os.WriteFile(doc, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, []string{"--dry-run", doc}, noEnv, loopback)
	if code != 2 || stdout != "" || !strings.Contains(stderr, "UTF-8") {
		t.Fatalf("invalid preview: %d %q %q", code, stdout, stderr)
	}
	p := func(string, string) (provider.Provider, error) {
		return reviewFunc(func(context.Context, provider.Request) (*provider.Result, error) {
			return &provider.Result{Findings: string([]byte{0xff})}, nil
		}), nil
	}
	code, stdout, stderr = runCLI(t, []string{"--json", "--provider", "test", controlTarget(t)}, noEnv, p)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "UTF-8") {
		t.Fatalf("invalid review output: %d %q %q", code, stdout, stderr)
	}
}
