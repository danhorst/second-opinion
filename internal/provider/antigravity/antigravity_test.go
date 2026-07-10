package antigravity

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

func TestBuildInvocationColdShape(t *testing.T) {
	req := provider.Request{Prompt: "find flaws", Material: "diff content"}
	argv := buildInvocation(req, "")

	if argv[0] != "agy" {
		t.Errorf("binary = %q", argv[0])
	}
	for _, flag := range []string{"--print", "--sandbox"} {
		if !slices.Contains(argv, flag) {
			t.Errorf("argv missing %s: %v", flag, argv)
		}
	}
	for _, forbidden := range []string{"--add-dir", "--dangerously-skip-permissions"} {
		if slices.Contains(argv, forbidden) {
			t.Errorf("argv must not carry %s: %v", forbidden, argv)
		}
	}
	if slices.Contains(argv, "--model") {
		t.Errorf("no forced model, but argv has --model: %v", argv)
	}

	instruction := argv[len(argv)-1]
	if !strings.HasPrefix(instruction, req.Prompt) {
		t.Errorf("instruction must start with the prompt: %q", instruction)
	}
	if !strings.Contains(instruction, req.Material) {
		t.Errorf("material must appear verbatim in the instruction")
	}
	if !strings.Contains(instruction, "--- MATERIAL") || !strings.Contains(instruction, "--- END MATERIAL ---") {
		t.Errorf("material must be delimited: %q", instruction)
	}
}

func TestBuildInvocationForcedModel(t *testing.T) {
	argv := buildInvocation(provider.Request{Prompt: "p"}, "Gemini 3.1 Pro (High)")
	i := slices.Index(argv, "--model")
	if i < 0 || argv[i+1] != "Gemini 3.1 Pro (High)" {
		t.Errorf("forced model missing from argv: %v", argv)
	}
}

func TestComposeInstructionDeterministic(t *testing.T) {
	req := provider.Request{Prompt: "p", Material: "m"}
	if composeInstruction(req) != composeInstruction(req) {
		t.Error("composition must be deterministic")
	}
}

func TestClassify(t *testing.T) {
	if err := classify(exec.ErrNotFound, "", ""); !errors.Is(err, provider.ErrUnavailable) {
		t.Errorf("missing binary must map to ErrUnavailable, got %v", err)
	}
	if err := classify(errors.New("exit status 1"), "Error: not signed in. Run agy to sign in.\n", ""); !errors.Is(err, provider.ErrAuth) {
		t.Errorf("sign-in failure must map to ErrAuth, got %v", err)
	}
	err := classify(errors.New("exit status 1"), "some other failure", "")
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
