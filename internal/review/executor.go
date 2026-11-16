package review

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/danhorst/second-opinion/internal/provider"
)

// ProviderFactory constructs one provider per call. It receives execution
// budgets separately from the content-only provider request.
type ProviderFactory func(name, model string, limits provider.CaptureLimits) (provider.Provider, error)

// Executor orchestrates one-shot reviews without retaining request state.
// The zero value uses the authoritative provider registry.
type Executor struct {
	NewProvider ProviderFactory
}

// ExecutionRequest contains all review inputs. Prompt omission selects the
// baked prompt; a supplied prompt and material are passed through byte-for-byte.
// Front-ends own their input validation and limits.
type ExecutionRequest struct {
	Provider      string
	Model         string
	Material      string
	Prompt        *string
	CaptureLimits provider.CaptureLimits
}

// ErrInvalidResult identifies a provider result that violates the contract.
var ErrInvalidResult = errors.New("review: invalid provider result")

// ResolvePrompt resolves explicit omission without normalizing supplied text.
func ResolvePrompt(prompt *string) string {
	if prompt == nil {
		return BakedPrompt
	}
	return *prompt
}

// Review constructs a fresh provider, propagates the caller context, and
// validates its result without formatting, normalizing, or ranking findings.
func (e Executor) Review(ctx context.Context, req ExecutionRequest) (*provider.Result, error) {
	factory := e.NewProvider
	if factory == nil {
		factory = NewProviderWithLimits
	}
	p, err := factory(req.Provider, req.Model, req.CaptureLimits)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("%w: factory returned no provider", ErrInvalidResult)
	}
	prompt := ResolvePrompt(req.Prompt)
	result, err := p.Review(ctx, provider.Request{Prompt: prompt, Material: req.Material})
	if result != nil && err != nil {
		return nil, fmt.Errorf("%w: result accompanied by error", ErrInvalidResult)
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("%w: missing result", ErrInvalidResult)
	}
	provenance := result.Provenance
	if strings.TrimSpace(provenance.Provider) == "" || strings.TrimSpace(provenance.Model) == "" || provenance.PromptHash != provider.HashPrompt(prompt) || provenance.MaterialHash != provider.HashMaterial(req.Material) {
		return nil, fmt.Errorf("%w: incomplete or inconsistent provenance", ErrInvalidResult)
	}
	if !utf8.ValidString(result.Findings) || !utf8.ValidString(provenance.Provider) || !utf8.ValidString(provenance.Model) {
		return nil, fmt.Errorf("%w: invalid UTF-8", ErrInvalidResult)
	}
	return result, nil
}
