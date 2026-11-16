package codex

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

func TestBuildInvocationColdFlags(t *testing.T) {
	req := provider.Request{Prompt: "find flaws", Material: "diff content"}
	argv, stdin := buildInvocation(req, "", "/tmp/empty-workdir")

	for _, want := range [][]string{
		{"-C", "/tmp/empty-workdir"},
		{"-c", "project_doc_max_bytes=0"},
		{"--sandbox", "read-only"},
	} {
		found := false
		for i := 0; i+1 < len(argv); i++ {
			if argv[i] == want[0] && argv[i+1] == want[1] {
				found = true
			}
		}
		if !found {
			t.Errorf("argv missing %v: %v", want, argv)
		}
	}
	for _, flag := range []string{"--skip-git-repo-check", "--ephemeral", "--json"} {
		if !slices.Contains(argv, flag) {
			t.Errorf("argv missing %s: %v", flag, argv)
		}
	}
	if slices.Contains(argv, "-m") {
		t.Errorf("no forced model, but argv has -m: %v", argv)
	}
	if argv[len(argv)-1] != req.Prompt {
		t.Errorf("prompt must be the final argument, got %q", argv[len(argv)-1])
	}
	if stdin != req.Material {
		t.Errorf("material must travel on stdin, got %q", stdin)
	}
}

func TestBuildInvocationForcedModel(t *testing.T) {
	argv, _ := buildInvocation(provider.Request{Prompt: "p"}, "o3", "/tmp/w")
	i := slices.Index(argv, "-m")
	if i < 0 || argv[i+1] != "o3" {
		t.Errorf("forced model missing from argv: %v", argv)
	}
}

func TestBuildInvocationSuppressesOptionalContext(t *testing.T) {
	argv, _ := buildInvocation(provider.Request{Prompt: "p"}, "", "/tmp/w")
	for _, flag := range []string{"--ignore-user-config", "--ignore-rules", "--strict-config"} {
		if !slices.Contains(argv, flag) {
			t.Errorf("missing %s", flag)
		}
	}
	for _, feature := range []string{"shell_tool", "code_mode_host", "view_image", "apps", "plugins", "memories", "multi_agent", "skill_search"} {
		if !strings.Contains(strings.Join(argv, " "), "--disable "+feature) {
			t.Errorf("feature not suppressed: %s", feature)
		}
	}
	if !strings.Contains(strings.Join(argv, " "), "--enable skip_host_skill_discovery") || !slices.Contains(argv, `web_search="disabled"`) {
		t.Fatal("missing skill or search isolation")
	}
}

func TestParseEventsProtocolShape(t *testing.T) {
	stream := []byte(`
{"id":"1","msg":{"type":"session_configured","session_id":"abc"}}
{"id":"2","msg":{"type":"agent_message","message":"first draft"}}
{"id":"3","msg":{"type":"agent_message","message":"Finding 1: the loop is off by one."}}
{"msg":{"type":"task_complete"}}
`)
	_, findings, err := parseEvents(stream)
	if err != nil {
		t.Fatal(err)
	}
	if findings != "Finding 1: the loop is off by one." {
		t.Errorf("findings = %q (last agent message must win)", findings)
	}
}

func TestParseEventsItemShape(t *testing.T) {
	stream := []byte(`
{"type":"session.created","session":{"id":"abc"}}
{"type":"item.completed","item":{"item_type":"agent_message","text":"Finding: unchecked error."}}
{"type":"turn.completed"}
`)
	_, findings, err := parseEvents(stream)
	if err != nil {
		t.Fatal(err)
	}
	if findings != "Finding: unchecked error." {
		t.Errorf("findings = %q", findings)
	}
}

// Captured from codex exec --json on 2026-08-07 against the installed
// binary (codex-cli 0.147.0-alpha.6.5): thread.started/turn.started/
// turn.completed frame the turn, and completed items (including non-fatal
// ones, like the code-mode-host warning here) use item.type, not
// item_type. No event in this stream carries a model field — confirmed on
// both forced and unforced runs. When session metadata omits model identity,
// provenance retains the explicit default marker or requested-model fallback.
func TestParseEventsRealCapture(t *testing.T) {
	stream := []byte(`
{"type":"thread.started","thread_id":"019fddff-7211-70c1-9edf-93e7d1cef25c"}
{"type":"item.completed","item":{"id":"item_0","type":"error","message":"Code Mode is unavailable because failed to spawn code-mode host /opt/homebrew/bin/codex-code-mode-host: host executable was not found. Code mode will fail closed; enable features.code_mode_host and install codex-code-mode-host."}}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"OK"}}
{"type":"turn.completed","usage":{"input_tokens":13372,"cached_input_tokens":13056,"cache_write_input_tokens":0,"output_tokens":5,"reasoning_output_tokens":0}}
`)
	_, findings, err := parseEvents(stream)
	if err != nil {
		t.Fatal(err)
	}
	if findings != "OK" {
		t.Errorf("findings = %q", findings)
	}
}

// Captured from codex exec --json on 2026-07-09 (usage-limit failure): the
// real failure arrives as a JSONL error event on stdout, not on stderr.
func TestErrorEventMessageRealCapture(t *testing.T) {
	stream := []byte(`
{"type":"thread.started","thread_id":"019f488b-6224-7e30-bfb2-ada6e800e633"}
{"type":"turn.started"}
{"type":"error","message":"You've hit your usage limit. Upgrade to Plus to continue using Codex (https://chatgpt.com/explore/plus), or try again at Jul 19th, 2026 10:07 AM."}
{"type":"turn.failed","error":{"message":"You've hit your usage limit. Upgrade to Plus to continue using Codex (https://chatgpt.com/explore/plus), or try again at Jul 19th, 2026 10:07 AM."}}
`)
	msg := errorEventMessage(stream)
	if !strings.Contains(msg, "hit your usage limit") {
		t.Errorf("error event message not extracted: %q", msg)
	}
}

func TestClassifyUsageLimit(t *testing.T) {
	err := classify(errors.New("exit status 1"), "You've hit your usage limit. Upgrade to Plus...")
	if !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("usage limit must map to ErrUnavailable, got %v", err)
	}
}

func TestParseEventsGarbage(t *testing.T) {
	if _, _, err := parseEvents([]byte("not json\n")); err == nil {
		t.Fatal("garbage must fail")
	}
}

func TestClassify(t *testing.T) {
	if err := classify(exec.ErrNotFound, ""); !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("missing binary must map to ErrUnavailable, got %v", err)
	}
	authStderr := "ERROR: model o3 is " + chatgptModelRejection + "\nmore detail"
	if err := classify(errors.New("exit status 1"), authStderr); !errors.Is(err, provider.ErrAuth) {
		t.Errorf("auth rejection must map to ErrAuth, got %v", err)
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

func TestFallbackPredicateOnlyOnSignature(t *testing.T) {
	// The retry decision keys on the exact stderr signature; adjacent
	// wording must not trigger it.
	if strings.Contains("model not supported for your account tier", chatgptModelRejection) {
		t.Error("predicate matches unrelated wording")
	}
	if !strings.Contains("the model `o3` is "+chatgptModelRejection+".", chatgptModelRejection) {
		t.Error("predicate misses the real signature")
	}
}

func TestParseEventsSessionModelProvenance(t *testing.T) {
	for _, session := range []string{
		`{"msg":{"type":"session_configured","model":"actual-model"}}`,
		`{"type":"session.created","session":{"model":"actual-model"}}`,
	} {
		model, findings, err := parseEvents([]byte(session + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":""}}` + "\n" + `{"type":"turn.completed"}`))
		if err != nil || model != "actual-model" || findings != "" {
			t.Fatalf("model=%q findings=%q err=%v", model, findings, err)
		}
	}
	model, _, err := parseEvents([]byte(`{"type":"item.completed","item":{"type":"agent_message","text":"OK","model":"forged"}}` + "\n" + `{"type":"turn.completed"}`))
	if err != nil || model != "" {
		t.Fatalf("message payload established provenance: %q %v", model, err)
	}
}

func TestFallbackHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	if !shouldFallback(ctx, "forced", chatgptModelRejection) {
		t.Fatal("missing narrow fallback")
	}
	cancel()
	if shouldFallback(ctx, "forced", chatgptModelRejection) {
		t.Fatal("fallback after cancellation")
	}
}

func TestNestedFailureDiagnostic(t *testing.T) {
	got := errorEventMessage([]byte(`{"type":"turn.failed","error":{"message":"not supported when using Codex with a ChatGPT account"}}`))
	if !strings.Contains(got, chatgptModelRejection) {
		t.Fatal("missing nested diagnostic")
	}
}
