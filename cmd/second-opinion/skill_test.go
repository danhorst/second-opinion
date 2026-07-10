package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/skills"
)

// The embed reads the repo file directly, so this pins the source-of-truth
// invariant and catches a stale build cache.
func TestEmbedMatchesRepo(t *testing.T) {
	b, err := os.ReadFile("../../skills/second-opinion/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != skills.SecondOpinion {
		t.Error("embedded skill differs from repository copy")
	}
}

func TestSkillContentEssentials(t *testing.T) {
	doc := skills.SecondOpinion
	for _, want := range []string{
		"--provider", "SECOND_OPINION_PROVIDER", "--diff", "--model", "--prompt-file",
		"reviewed-by:", "severity × novelty", "two-thirds",
		"same model defeats the premise",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("skill missing %q", want)
		}
	}
	front, _ := splitFrontmatter(doc)
	if !strings.Contains(front, "name: second-opinion") {
		t.Errorf("frontmatter missing name: %q", front)
	}
}

type fakeImport struct {
	ran      bool
	dir      string
	agyFound bool
}

func (f *fakeImport) deps(home string) skillDeps {
	return skillDeps{
		home: home,
		lookPath: func(string) (string, error) {
			if f.agyFound {
				return "/fake/agy", nil
			}
			return "", errors.New("not found")
		},
		runInstall: func(pluginDir string) (string, error) {
			f.ran = true
			f.dir = pluginDir
			return "ok", nil
		},
	}
}

func TestSkillInstallDefault(t *testing.T) {
	home := t.TempDir()
	fake := &fakeImport{agyFound: true}
	var out, errBuf strings.Builder

	if code := runSkill([]string{"install"}, fake.deps(home), &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d (stderr: %s)", code, errBuf.String())
	}

	claudePath := filepath.Join(home, ".claude", "skills", "second-opinion", "SKILL.md")
	codexPath := filepath.Join(home, ".codex", "prompts", "second-opinion.md")

	claude, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(claude), "---\n") || !strings.Contains(string(claude), generatedStamp) {
		t.Error("claude install must keep frontmatter and carry the stamp")
	}

	codex, err := os.ReadFile(codexPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(codex), "name: second-opinion") {
		t.Error("codex install must strip frontmatter")
	}
	if !strings.Contains(string(codex), "severity × novelty") {
		t.Error("codex install must keep the body")
	}

	if !fake.ran {
		t.Error("agy present: plugin install must run")
	}
	wantDir := filepath.Join(home, ".second-opinion", "antigravity-plugin")
	if fake.dir != wantDir {
		t.Errorf("plugin dir = %q, want %q", fake.dir, wantDir)
	}
	if _, err := os.Stat(filepath.Join(wantDir, "plugin.json")); err != nil {
		t.Errorf("plugin.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wantDir, "skills", "second-opinion", "SKILL.md")); err != nil {
		t.Errorf("plugin skill missing: %v", err)
	}
}

func TestSkillInstallNoAgy(t *testing.T) {
	home := t.TempDir()
	fake := &fakeImport{agyFound: false}
	var out, errBuf strings.Builder

	if code := runSkill([]string{"install"}, fake.deps(home), &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if fake.ran {
		t.Error("agy absent: plugin install must not run")
	}
	if !strings.Contains(out.String(), "agy plugin install") {
		t.Errorf("must print the install as a next step: %q", out.String())
	}
	// The plugin dir is still written, so the printed command works as-is.
	if _, err := os.Stat(filepath.Join(home, ".second-opinion", "antigravity-plugin", "plugin.json")); err != nil {
		t.Errorf("plugin dir must be written even without agy: %v", err)
	}
}

func TestSkillInstallHarnessSelection(t *testing.T) {
	home := t.TempDir()
	fake := &fakeImport{agyFound: false}
	var out, errBuf strings.Builder

	if code := runSkill([]string{"install", "--harness", "codex"}, fake.deps(home), &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); !os.IsNotExist(err) {
		t.Error("codex-only install must not touch ~/.claude")
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "prompts", "second-opinion.md")); err != nil {
		t.Errorf("codex file missing: %v", err)
	}
}

func TestSkillInstallIdempotentAndSurgical(t *testing.T) {
	home := t.TempDir()
	fake := &fakeImport{}
	var out, errBuf strings.Builder

	unrelated := filepath.Join(home, ".codex", "prompts", "mine.md")
	os.MkdirAll(filepath.Dir(unrelated), 0o755)
	os.WriteFile(unrelated, []byte("hands off"), 0o644)

	for range 2 {
		if code := runSkill([]string{"install"}, fake.deps(home), &out, &errBuf); code != 0 {
			t.Fatalf("exit = %d", code)
		}
	}
	if b, _ := os.ReadFile(unrelated); string(b) != "hands off" {
		t.Error("unrelated file was touched")
	}
}

func TestSkillStdout(t *testing.T) {
	home := t.TempDir()
	fake := &fakeImport{agyFound: true}
	var out, errBuf strings.Builder

	if code := runSkill([]string{"install", "--stdout"}, fake.deps(home), &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if out.String() != skills.SecondOpinion {
		t.Error("--stdout must print the canonical skill")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); !os.IsNotExist(err) {
		t.Error("--stdout must not write files")
	}
	if fake.ran {
		t.Error("--stdout must not run the import")
	}
}

func TestSkillUsageErrors(t *testing.T) {
	fake := &fakeImport{}
	for _, args := range [][]string{{}, {"remove"}, {"install", "--harness"}, {"install", "--harness", "vim"}, {"install", "--bogus"}} {
		var out, errBuf strings.Builder
		if code := runSkill(args, fake.deps(t.TempDir()), &out, &errBuf); code != 2 {
			t.Errorf("args %v: exit = %d, want 2", args, code)
		}
	}
}
