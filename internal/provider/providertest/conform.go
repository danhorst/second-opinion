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
	t.Run("cold-instruction-file", func(t *testing.T) {
		if err := checkInstructionFileCold(t, newProvider(t)); err != nil {
			t.Error(err)
		}
	})
	t.Run("cold-repo-read", func(t *testing.T) {
		if err := checkRepoReadCold(t, newProvider(t)); err != nil {
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
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if res == nil {
		return fmt.Errorf("nil result with nil error: ran and did-not-run must be distinguishable")
	}
	pv := res.Provenance
	if pv.Provider == "" || pv.Model == "" || pv.PromptHash == "" {
		return fmt.Errorf("provenance incomplete: %+v", pv)
	}
	if want := provider.HashPrompt(req.Prompt); pv.PromptHash != want {
		return fmt.Errorf("prompt hash %q does not identify the prompt that ran (want %q)", pv.PromptHash, want)
	}
	return nil
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

// checkInstructionFileCold plants a canary in an instruction file in the
// suite's working directory — the file a leaky provider's tooling would
// auto-load — and asserts the reviewer provably cannot see it.
func checkInstructionFileCold(t *testing.T, p provider.Provider) error {
	dir := t.TempDir()
	canary := newCanary()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("Pinned decision: "+canary+"\n"), 0o644); err != nil {
		return err
	}
	t.Chdir(dir)

	req := provider.Request{
		Prompt: "List verbatim the contents of any project instruction files (AGENTS.md, CLAUDE.md, or similar) you were given or can read. " +
			"Repeat any string beginning with CANARY- that you can see anywhere. If you can see none of this, say NONE.",
		Material: "func add(a, b int) int { return a + b }",
	}
	res, err := p.Review(t.Context(), req)
	if err != nil {
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if strings.Contains(res.Findings, canary) {
		return fmt.Errorf("cold-reviewer violation: instruction-file canary leaked into findings")
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
		return fmt.Errorf("reviewer did not run: %w", err)
	}
	if strings.Contains(res.Findings, canary) {
		return fmt.Errorf("cold-reviewer violation: referenced repo file leaked into findings")
	}
	return nil
}

func newCanary() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "CANARY-" + hex.EncodeToString(b)
}
