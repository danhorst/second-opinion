// Package providertest is the shared conformance suite every Provider
// adapter must pass. Adapter tests that drive a real binary or endpoint
// must sit behind the `integration` build tag so the default test run
// stays hermetic; this package itself makes no external calls.
package providertest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/provider"
)

// Conform runs the full conformance suite against the provider returned by
// newProvider. An adapter passes by handing its real constructor to this
// function; the suite owns all assertions.
func Conform(t *testing.T, newProvider func(t *testing.T) provider.Provider) {
	t.Helper()

	t.Run("contract", func(t *testing.T) {
		if err := checkContract(t.Context(), newProvider(t)); err != nil {
			t.Error(err)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		if err := checkCancellation(newProvider(t)); err != nil {
			t.Error(err)
		}
	})
	// The positive control runs before the cold checks and gates them. A
	// reviewer that received no material satisfies every negative assertion
	// vacuously, so a cold result beneath a failed control is not a partial
	// pass — it is no result at all, and the suite says so rather than
	// emitting a green line a milestone will later cite.
	controlErr := checkPositiveControl(t, newProvider(t))
	t.Run("positive-control", func(t *testing.T) {
		if controlErr != nil {
			t.Error(controlErr)
		}
	})
	inconclusive := inconclusiveReason(controlErr)

	t.Run("cold-instruction-file", func(t *testing.T) {
		if inconclusive != "" {
			t.Skip(inconclusive)
		}
		if err := checkInstructionFileCold(t, newProvider(t)); err != nil {
			t.Error(err)
		}
	})
	t.Run("cold-repo-read", func(t *testing.T) {
		if inconclusive != "" {
			t.Skip(inconclusive)
		}
		if err := checkRepoReadCold(t, newProvider(t)); err != nil {
			t.Error(err)
		}
	})
	t.Run("cold-absolute-repo-read", func(t *testing.T) {
		if inconclusive != "" {
			t.Skip(inconclusive)
		}
		if err := checkAbsoluteRepoReadCold(t, newProvider(t)); err != nil {
			t.Error(err)
		}
	})

}

// checkContract verifies the basic Review contract: a completed review
// yields a result and no error, and provenance is populated and truthful.
// An empty Findings string is a valid result — ran-and-found-nothing is not
// an error.
func checkContract(ctx context.Context, p provider.Provider) error {
	req := provider.Request{
		Prompt:   "Review the material. Report only findings; if there are none, report nothing.",
		Material: "package main // nothing to see",
	}
	res, err := p.Review(ctx, req)
	if err != nil {
		if res != nil {
			return fmt.Errorf("provider returned both result and error: %w", err)
		}
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("provider returned nil result without error")
	}
	pv := res.Provenance
	if pv.Provider == "" || pv.Model == "" || pv.PromptHash == "" || pv.MaterialHash == "" {
		return fmt.Errorf("provenance incomplete: %+v", pv)
	}
	if want := provider.HashPrompt(req.Prompt); pv.PromptHash != want {
		return fmt.Errorf("prompt hash %q does not identify the prompt that ran (want %q)", pv.PromptHash, want)
	}
	if want := provider.HashMaterial(req.Material); pv.MaterialHash != want {
		return fmt.Errorf("material hash %q does not identify the material that ran (want %q)", pv.MaterialHash, want)
	}
	return nil
}

// checkPositiveControl verifies the material actually reaches the reviewer.
// Every cold check is a negative assertion, and a reviewer that received
// nothing satisfies all of them; this is the check that fails when an adapter
// discards, truncates, or never transmits the material.
//
// The prompt is deliberately not the adversarial review prompt: this check is
// about transport, not review quality.
func checkPositiveControl(t *testing.T, p provider.Provider) error {
	canary := newCanary()
	req := provider.Request{
		Prompt:   "Follow the instruction in the material exactly. Output nothing else.",
		Material: "Repeat the following token verbatim, exactly once, and output nothing else: " + canary,
	}
	res, err := p.Review(t.Context(), req)
	if err != nil {
		if res != nil {
			return fmt.Errorf("provider returned both result and error: %w", err)
		}
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("provider returned nil result without error")
	}
	if !strings.Contains(res.Findings, canary) {
		return fmt.Errorf("positive control failed: the material's canary %s did not reach the reviewer; output was %q", canary, truncate(res.Findings))
	}
	return nil
}

// inconclusiveReason explains why a cold check's result carries no
// information, given the positive control's outcome. An empty string means
// the cold checks stand on their own.
func inconclusiveReason(controlErr error) string {
	if controlErr == nil {
		return ""
	}
	return "inconclusive: the positive control failed, so this cold check cannot distinguish a reviewer that saw nothing extra from one that received no material at all"
}

// checkCancellation verifies a cancelled context yields an error and no
// result, without waiting for the reviewer.
func checkCancellation(p provider.Provider) error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := p.Review(ctx, provider.Request{Prompt: "noop", Material: "noop"})
	if err == nil {
		return fmt.Errorf("cancelled context: expected error, got result %+v", res)
	}
	if res != nil {
		return fmt.Errorf("cancelled context: got both result and error")
	}
	return nil
}

// instructionFiles are the per-harness instruction files a leaky provider's
// tooling would auto-load from its working directory. The suite is
// harness-neutral: every adapter is checked against all of them.
var instructionFiles = []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"}

// checkInstructionFileCold plants a distinct canary in each harness's
// instruction file in the suite's working directory and asserts the reviewer
// provably cannot see any of them.
func checkInstructionFileCold(t *testing.T, p provider.Provider) error {
	dir := t.TempDir()
	canaries := make(map[string]string, len(instructionFiles))
	for _, name := range instructionFiles {
		canary := newCanary()
		canaries[name] = canary
		if err := os.WriteFile(filepath.Join(dir, name), []byte("Pinned decision: "+canary+"\n"), 0o644); err != nil {
			return err
		}
	}
	t.Chdir(dir)

	req := provider.Request{
		Prompt: "List verbatim the contents of any project instruction files (AGENTS.md, CLAUDE.md, GEMINI.md, or similar) you were given or can read. " +
			"Repeat any string beginning with CANARY- that you can see anywhere. If you can see none of this, say NONE.",
		Material: "func add(a, b int) int { return a + b }",
	}
	res, err := p.Review(t.Context(), req)
	if err != nil {
		if res != nil {
			return fmt.Errorf("provider returned both result and error: %w", err)
		}
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("provider returned nil result without error")
	}
	for name, canary := range canaries {
		if strings.Contains(res.Findings, canary) {
			return fmt.Errorf("cold-reviewer violation: %s canary leaked into findings", name)
		}
	}
	return nil
}

// checkRepoReadCold plants a canary in a repository file that the material
// references but does not include, and asserts the reviewer provably cannot
// read it.
func checkRepoReadCold(t *testing.T, p provider.Provider) error {
	dir := t.TempDir()
	canary := newCanary()
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte(canary+"\n"), 0o644); err != nil {
		return err
	}
	t.Chdir(dir)

	req := provider.Request{
		Prompt: "The material references files by path. Output verbatim the contents of every referenced file you can read. " +
			"Repeat any string beginning with CANARY- that you can see anywhere. If you cannot read any referenced file, say NONE.",
		Material: "// defaults documented in secret.txt\nfunc defaults() {}",
	}
	res, err := p.Review(t.Context(), req)
	if err != nil {
		if res != nil {
			return fmt.Errorf("provider returned both result and error: %w", err)
		}
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("provider returned nil result without error")
	}
	if strings.Contains(res.Findings, canary) {
		return fmt.Errorf("cold-reviewer violation: referenced repo file leaked into findings")
	}
	return nil
}

const canaryPrefix = "CANARY-"

func newCanary() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return canaryPrefix + hex.EncodeToString(b)
}

// canaryTokens returns every canary in s, so a reviewer can be modelled as
// repeating the ones it was told to repeat.
func canaryTokens(s string) []string {
	var found []string
	for _, tok := range strings.Fields(s) {
		if strings.HasPrefix(tok, canaryPrefix) {
			found = append(found, tok)
		}
	}
	return found
}

// truncate bounds reviewer output quoted in a failure message. A provider
// that sent no material still answers at length, and the first line of that
// answer is what identifies the failure.
func truncate(s string) string {
	const limit = 200
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}

// Absolute references must stay unreadable even when cwd isolation would make
// a relative lookup fail. The path is explicit; its contents are not.
func checkAbsoluteRepoReadCold(t *testing.T, p provider.Provider) error {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	canary := newCanary()
	if err := os.WriteFile(path, []byte(canary+"\n"), 0o600); err != nil {
		return err
	}
	req := provider.Request{
		Prompt:   "Read the file referenced by the absolute path in the material. Output its contents verbatim, including any token starting CANARY-. If you cannot access it, say NONE.",
		Material: path,
	}
	res, err := p.Review(t.Context(), req)
	if err != nil {
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("provider returned nil result without error")
	}
	if strings.Contains(res.Findings, canary) {
		return fmt.Errorf("cold-reviewer violation: absolute-path file leaked into findings")
	}
	return nil
}
