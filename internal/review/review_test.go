package review

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

// The baked prompt must not presume any transport — these are the phrases
// each harness's delivery mechanism would tempt into it.
func TestBakedPromptTransportNeutral(t *testing.T) {
	for _, phrase := range []string{"<stdin>", "stdin", "MATERIAL block", "file path", "attached file"} {
		if strings.Contains(strings.ToLower(BakedPrompt), strings.ToLower(phrase)) {
			t.Errorf("baked prompt references a transport: %q", phrase)
		}
	}
}

func TestFromFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("alpha"), 0o644)
	os.WriteFile(b, []byte("beta"), 0o644)

	material, err := FromFiles([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	ia, ib := strings.Index(material, "alpha"), strings.Index(material, "beta")
	if ia < 0 || ib < 0 || ia > ib {
		t.Errorf("files must appear in argument order: %q", material)
	}
	if !strings.Contains(material, "=== "+a+" ===") {
		t.Errorf("each file needs a path header: %q", material)
	}
}

func TestFromFilesMissing(t *testing.T) {
	if _, err := FromFiles([]string{filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Error("missing file must error")
	}
}

func TestFromFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.txt")
	os.WriteFile(empty, nil, 0o644)
	if _, err := FromFiles([]string{empty}); !errors.Is(err, ErrNothingToReview) {
		t.Errorf("empty material must be refused, got %v", err)
	}
}

func TestFromDiff(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	os.WriteFile("f.txt", []byte("one\n"), 0o644)
	run("add", "f.txt")
	run("commit", "-q", "-m", "base")

	if _, err := FromDiff("HEAD"); !errors.Is(err, ErrNothingToReview) {
		t.Errorf("clean tree must refuse, got %v", err)
	}

	os.WriteFile("f.txt", []byte("two\n"), 0o644)
	material, err := FromDiff("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(material, "-one") || !strings.Contains(material, "+two") {
		t.Errorf("diff content missing: %q", material)
	}

	// Exercise actual Git configuration, not a stub executable. A helper
	// would fail if invoked; a successful patch proves it was bypassed.
	run("config", "color.ui", "always")
	t.Setenv("GIT_EXTERNAL_DIFF", "second-opinion-nonexistent-helper")
	run("config", "diff.test.textconv", "second-opinion-nonexistent-helper")
	if err := os.WriteFile(".gitattributes", []byte("*.txt diff=test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configured, err := FromDiff("HEAD")
	if err != nil || configured != material {
		t.Fatalf("Git customization altered patch: %v\n%s", err, configured)
	}
	for _, base := range []string{"f.txt", "--stat", ""} {
		if _, err := FromDiff(base); err == nil || errors.Is(err, ErrNothingToReview) {
			t.Errorf("base %q must fail, got %v", base, err)
		}
	}
	// A branch and a file can share a name; the revision must win.
	run("branch", "f.txt")
	if got, err := FromDiff("f.txt"); err != nil || got != material {
		t.Errorf("ambiguous revision: %v\n%s", err, got)
	}
	run("add", "f.txt")
	if err := os.WriteFile("f.txt", []byte("three\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("untracked.txt", []byte("untracked-canary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := FromDiff("HEAD"); err != nil || !strings.Contains(got, "-one") || !strings.Contains(got, "+three") || strings.Contains(got, "untracked-canary") {
		t.Errorf("tracked staged and unstaged diff: %v\n%s", err, got)
	}
}

func TestFromDiffCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FromDiffContext(ctx, "HEAD"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled assembly: %v", err)
	}
}

func TestFromDiffNotARepo(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := FromDiff("HEAD"); err == nil || errors.Is(err, ErrNothingToReview) {
		t.Errorf("outside a repo must be a hard error, got %v", err)
	}
}

func TestRegistry(t *testing.T) {
	want := []string{"antigravity", "claude", "codex"}
	got := Providers()
	if len(got) != len(want) {
		t.Fatalf("providers = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("providers = %v, want %v", got, want)
		}
	}
	for _, name := range want {
		if _, err := NewProvider(name, ""); err != nil {
			t.Errorf("NewProvider(%s): %v", name, err)
		}
		if _, err := NewProvider(name, "some-model"); err != nil {
			t.Errorf("NewProvider(%s, model): %v", name, err)
		}
	}
	if _, err := NewProvider("nope", ""); err == nil || !strings.Contains(err.Error(), "antigravity") {
		t.Errorf("unknown provider must error naming the registered set, got %v", err)
	}
}

// Exercise the real adapter entry points with genuinely absent executables.
// No fake executable stands in for a provider and no network call is made.
func TestUnavailableProviderBinaries(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, name := range Providers() {
		t.Run(name, func(t *testing.T) {
			p, err := NewProvider(name, "")
			if err != nil {
				t.Fatal(err)
			}
			res, err := p.Review(t.Context(), provider.Request{Prompt: "review", Material: "material"})
			if res != nil || !errors.Is(err, provider.ErrUnavailable) {
				t.Fatalf("absent binary: result=%+v error=%v", res, err)
			}
		})
	}
}
