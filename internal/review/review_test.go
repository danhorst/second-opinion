package review

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
