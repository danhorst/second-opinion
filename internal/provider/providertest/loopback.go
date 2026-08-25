package providertest

import (
	"context"
	"os"
	"strings"

	"github.com/danhorst/second-opinion/internal/provider"
)

// Loopback is the in-process reference provider. Its only purpose is to
// exercise the conformance suite: honest by default, riggable to commit
// each violation the suite must catch. It is not a stand-in for a real
// adapter in adapter tests.
type Loopback struct {
	leakInstructionFile bool
	readReferencedFiles bool
	omitProvenance      bool
	ignoreCancellation  bool
	unavailable         bool
}

// LoopbackOption rigs a Loopback to violate the contract.
type LoopbackOption func(*Loopback)

// LeakInstructionFile makes the provider read every harness's instruction
// file from its working directory into the findings, imitating tooling that
// auto-loads project instruction files.
func LeakInstructionFile() LoopbackOption {
	return func(l *Loopback) { l.leakInstructionFile = true }
}

// ReadReferencedFiles makes the provider dereference file paths mentioned
// in the material, imitating a reviewer with repository access.
func ReadReferencedFiles() LoopbackOption {
	return func(l *Loopback) { l.readReferencedFiles = true }
}

// OmitProvenance makes the provider return a result with empty provenance.
func OmitProvenance() LoopbackOption {
	return func(l *Loopback) { l.omitProvenance = true }
}

// IgnoreCancellation makes the provider return a result even when the
// context is cancelled.
func IgnoreCancellation() LoopbackOption {
	return func(l *Loopback) { l.ignoreCancellation = true }
}

// Unavailable makes every review fail with ErrUnavailable, for exercising
// the did-not-run semantics.
func Unavailable() LoopbackOption {
	return func(l *Loopback) { l.unavailable = true }
}

// NewLoopback returns a reference provider, honest unless rigged.
func NewLoopback(opts ...LoopbackOption) *Loopback {
	l := &Loopback{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Review implements provider.Provider. The honest path sees exactly the
// request and nothing else, so it finds nothing.
func (l *Loopback) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	if l.unavailable {
		return nil, provider.ErrUnavailable
	}
	if !l.ignoreCancellation {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	var leaked []string
	if l.leakInstructionFile {
		for _, name := range instructionFiles {
			if b, err := os.ReadFile(name); err == nil {
				leaked = append(leaked, string(b))
			}
		}
	}
	if l.readReferencedFiles {
		for _, tok := range strings.Fields(req.Material) {
			if b, err := os.ReadFile(tok); err == nil {
				leaked = append(leaked, string(b))
			}
		}
	}

	res := &provider.Result{Findings: strings.Join(leaked, "\n")}
	if !l.omitProvenance {
		res.Provenance = provider.Provenance{
			Provider:     "loopback",
			Model:        "loopback-1",
			PromptHash:   provider.HashPrompt(req.Prompt),
			MaterialHash: provider.HashMaterial(req.Material),
		}
	}
	return res, nil
}
