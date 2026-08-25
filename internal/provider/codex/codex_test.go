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
		i := slices.Index(argv, want[0])
		if i < 0 || i+1 >= len(argv) || argv[i+1] != want[1] {
			t.Errorf("argv missing %v: %v", want, argv)
		}
	}
	for _, flag := range []string{"--ignore-user-config", "--ignore-rules", "--skip-git-repo-check", "--ephemeral", "--json"} {
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

func TestParseEventsProtocolShape(t *testing.T) {
	stream := []byte(`
{"id":"1","msg":{"type":"session_configured","model":"gpt-5.3-codex","session_id":"abc"}}
{"id":"2","msg":{"type":"agent_message","message":"first draft"}}
{"id":"3","msg":{"type":"agent_message","message":"Finding 1: the loop is off by one."}}
`)
	model, findings, hasMessage := parseEvents(stream)
	if model != "gpt-5.3-codex" {
		t.Errorf("model = %q", model)
	}
	if findings != "Finding 1: the loop is off by one." {
		t.Errorf("findings = %q (last agent message must win)", findings)
	}
	if !hasMessage {
		t.Error("agent message must be recognized")
	}
}

func TestParseEventsItemShape(t *testing.T) {
	stream := []byte(`
{"type":"session.created","session":{"model":"gpt-5.3-codex"}}
{"type":"item.completed","item":{"type":"agent_message","text":"Finding: unchecked error."}}
`)
	model, findings, hasMessage := parseEvents(stream)
	if model != "gpt-5.3-codex" {
		t.Errorf("model = %q", model)
	}
	if findings != "Finding: unchecked error." {
		t.Errorf("findings = %q", findings)
	}
	if !hasMessage {
		t.Error("agent message must be recognized")
	}
}

func TestParseEventsRecognizesEmptyAgentMessage(t *testing.T) {
	model, findings, hasMessage := parseEvents([]byte(`{"model":"gpt-5.3-codex"}
{"type":"agent_message","message":""}`))
	if model != "gpt-5.3-codex" || findings != "" || !hasMessage {
		t.Errorf("empty agent message must be recognized, got %q / %q / %t", model, findings, hasMessage)
	}
}

func TestDefaultModelMarker(t *testing.T) {
	if defaultModelMarker != "codex-default" {
		t.Errorf("default model marker = %q", defaultModelMarker)
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
	model, findings, hasMessage := parseEvents([]byte("not json\n{\"broken\":\ntext\n"))
	if model != "" || findings != "" || hasMessage {
		t.Errorf("garbage stream must parse to nothing, got %q / %q / %t", model, findings, hasMessage)
	}
}

func TestErrorEventMessageNestedCapture(t *testing.T) {
	stream := []byte(`{"type":"turn.failed","error":{"message":"the model is not supported when using Codex with a ChatGPT account"}}`)
	if got := errorEventMessage(stream); !strings.Contains(got, chatgptModelRejection) {
		t.Errorf("nested error message not extracted: %q", got)
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

func TestReviewDoesNotFallbackAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if shouldFallback(ctx, "o3", chatgptModelRejection) {
		t.Error("cancelled context must not trigger fallback")
	}
}

func TestFallbackRequiresModelRejectionAndFailure(t *testing.T) {
	ctx := context.Background()
	if !shouldFallback(ctx, "o3", chatgptModelRejection) {
		t.Error("forced model rejection must trigger fallback predicate")
	}
	if shouldFallback(ctx, "o3", "some other failure") {
		t.Error("unrelated failure must not trigger fallback predicate")
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
