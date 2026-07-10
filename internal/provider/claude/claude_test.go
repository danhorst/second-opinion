package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

func TestBuildInvocationColdFlags(t *testing.T) {
	req := provider.Request{Prompt: "find flaws", Material: "diff content"}
	argv := buildInvocation(req, "")

	if argv[0] != "claude" || argv[1] != "-p" {
		t.Errorf("must invoke claude print mode: %v", argv[:2])
	}
	// The suppression pairs the cold guarantee depends on, including the
	// harness-global-context suppression required by provider-conformance.
	for _, want := range [][]string{
		{"--output-format", "json"},
		{"--tools", ""},
		{"--setting-sources", ""},
	} {
		i := slices.Index(argv, want[0])
		if i < 0 || i+1 >= len(argv) || argv[i+1] != want[1] {
			t.Errorf("argv missing %q %q: %v", want[0], want[1], argv)
		}
	}
	for _, forbidden := range []string{"--add-dir", "--dangerously-skip-permissions", "--allow-dangerously-skip-permissions"} {
		if slices.Contains(argv, forbidden) {
			t.Errorf("argv must not carry %s: %v", forbidden, argv)
		}
	}
	if slices.Contains(argv, "--model") {
		t.Errorf("no forced model, but argv has --model: %v", argv)
	}

	instruction := argv[len(argv)-1]
	if !strings.HasPrefix(instruction, req.Prompt) || !strings.Contains(instruction, req.Material) {
		t.Errorf("instruction must carry prompt and material verbatim: %q", instruction)
	}
	if !strings.Contains(instruction, "--- MATERIAL") || !strings.Contains(instruction, "--- END MATERIAL ---") {
		t.Errorf("material must be delimited: %q", instruction)
	}
}

func TestBuildInvocationForcedModel(t *testing.T) {
	argv := buildInvocation(provider.Request{Prompt: "p"}, "opus")
	i := slices.Index(argv, "--model")
	if i < 0 || argv[i+1] != "opus" {
		t.Errorf("forced model missing from argv: %v", argv)
	}
}

// Captured from claude CLI 2.1.206 on 2026-07-10, trimmed to the fields the
// adapter reads. The haiku entry is a harness helper; sonnet wrote the
// findings.
const probeEnvelope = `{"type":"result","subtype":"success","is_error":false,
"result":"NOSTDIN\nNOMEMORY\nI'm Claude Sonnet 5 (model ID: claude-sonnet-5).",
"modelUsage":{
 "claude-haiku-4-5-20251001":{"inputTokens":604,"outputTokens":12},
 "claude-sonnet-5":{"inputTokens":1,"outputTokens":550}}}`

func TestEnvelopeParsingProbeFixture(t *testing.T) {
	var env envelope
	if err := json.Unmarshal([]byte(probeEnvelope), &env); err != nil {
		t.Fatalf("fixture must parse: %v", err)
	}
	if env.IsError {
		t.Error("fixture is a success envelope")
	}
	if got := env.dominantModel(); got != "claude-sonnet-5" {
		t.Errorf("dominant model = %q; helper model must not win", got)
	}
	if !strings.Contains(env.Result, "NOMEMORY") {
		t.Errorf("result not extracted: %q", env.Result)
	}
}

func TestDominantModelEmpty(t *testing.T) {
	if got := (envelope{}).dominantModel(); got != "" {
		t.Errorf("empty usage must yield empty model, got %q", got)
	}
}

func TestClassify(t *testing.T) {
	if err := classify(exec.ErrNotFound, ""); !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("missing binary must map to ErrUnavailable, got %v", err)
	}
	if err := classify(errors.New("exit status 1"), "Invalid API key. Please run /login"); !errors.Is(err, provider.ErrAuth) {
		t.Errorf("auth failure must map to ErrAuth, got %v", err)
	}
	if err := classify(errors.New("exit status 1"), "You've reached your usage limit for today"); !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("usage limit must map to ErrUnavailable, got %v", err)
	}
	err := classify(errors.New("exit status 1"), "some other failure")
	if errors.Is(err, provider.ErrAuth) || errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("unrelated failure must not map to a sentinel, got %v", err)
	}
}

func TestReviewPreCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := New().Review(ctx, provider.Request{Prompt: "p", Material: "m"})
	if err == nil || res != nil {
		t.Errorf("pre-cancelled context: want error and nil result, got %v / %+v", err, res)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}
