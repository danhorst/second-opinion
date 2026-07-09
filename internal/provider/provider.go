// Package provider defines the contract between the review engine and any
// reviewer: a request of explicitly assembled material in, findings with
// reviewer provenance out.
package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Request is the entire input to a review. Material is passed by value —
// there is deliberately no way to hand a provider a path, repository
// reference, or URL it would dereference itself.
type Request struct {
	// Prompt is the review prompt, fully rendered.
	Prompt string
	// Material is the content under review, fully assembled by the caller.
	Material string
}

// Provenance identifies who performed a review, so that a
// reviewer-equals-author violation is detectable after the fact.
type Provenance struct {
	// Provider is the adapter name, e.g. "codex".
	Provider string
	// Model is the model that actually ran — not the one requested.
	Model string
	// PromptHash is HashPrompt of the request's Prompt.
	PromptHash string
}

// Result is a completed review. Findings is raw reviewer output; its
// structure is deliberately undecided, which is why Provenance travels
// outside it.
type Result struct {
	Findings   string
	Provenance Provenance
}

// Provider performs adversarial reviews. Review returns a Result when the
// reviewer ran — empty Findings mean it ran and found nothing — and an error
// when it did not run. Never both.
type Provider interface {
	Review(ctx context.Context, req Request) (*Result, error)
}

// Sentinel errors classifying why a reviewer did not run.
var (
	ErrUnavailable = errors.New("provider: reviewer unavailable")
	ErrAuth        = errors.New("provider: authentication rejected")
)

// HashPrompt computes the canonical prompt identity used in Provenance.
// All adapters use this helper so prompt identity cannot diverge.
func HashPrompt(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}
