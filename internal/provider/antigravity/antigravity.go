// Package antigravity implements provider.Provider by shelling out to the
// antigravity CLI (agy). Cold by construction: the process runs from an
// empty temp directory with sandbox restrictions and no workspace grants,
// and the material travels by value inside the composed instruction —
// agy's print mode does not read stdin.
package antigravity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/danhorst/second-opinion/internal/provider"
)

const providerName = "antigravity"

// defaultModelMarker is the documented provenance degradation for unforced
// reviews: agy has no structured output naming the model that ran, so an
// unforced review reports this marker rather than an invented model name.
const defaultModelMarker = "antigravity-default"

// authFailureSignatures classify a did-not-run as ErrAuth. Extended as
// integration runs surface real wording.
var authFailureSignatures = []string{
	"not signed in",
	"sign in",
	"authentication",
	"unauthorized",
}

// Provider runs reviews through the antigravity CLI.
type Provider struct {
	model string // forced model; empty means agy's default
}

// Option configures a Provider.
type Option func(*Provider)

// WithModel forces a model (passed to agy as --model). Provenance names it;
// agy either honors the name or fails loudly.
func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

// New returns an antigravity-backed Provider.
func New(opts ...Option) *Provider {
	p := &Provider{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// composeInstruction merges the prompt and material into the single
// instruction agy receives. Pure and deterministic: what the reviewer saw
// is reproducible from the request.
func composeInstruction(req provider.Request) string {
	return req.Prompt +
		"\n\n--- MATERIAL (verbatim, assembled by the caller) ---\n" +
		req.Material +
		"\n--- END MATERIAL ---\n"
}

// buildInvocation constructs the agy command line. Pure so the cold
// invocation is testable without executing agy. Isolation comes from the
// caller running the process with its working directory set to workdir;
// there is no stdin.
func buildInvocation(req provider.Request, model string) (argv []string) {
	argv = []string{"agy", "--print", "--sandbox"}
	if model != "" {
		argv = append(argv, "--model", model)
	}
	argv = append(argv, composeInstruction(req))
	return argv
}

// Review implements provider.Provider.
func (p *Provider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workdir, err := os.MkdirTemp("", "second-opinion-antigravity-")
	if err != nil {
		return nil, fmt.Errorf("antigravity: creating workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	argv := buildInvocation(req, p.model)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = workdir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, classify(err, stderr.String(), stdout.String())
	}

	model := p.model
	if model == "" {
		model = defaultModelMarker
	}
	return &provider.Result{
		Findings: strings.TrimSpace(stdout.String()),
		Provenance: provider.Provenance{
			Provider:   providerName,
			Model:      model,
			PromptHash: provider.HashPrompt(req.Prompt),
		},
	}, nil
}

// classify maps an agy failure onto the contract's did-not-run sentinels.
func classify(err error, stderr, stdout string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: agy binary not found: %v", provider.ErrUnavailable, err)
	}
	diag := firstLine(stderr)
	if diag == "" {
		diag = firstLine(stdout)
	}
	lower := strings.ToLower(diag)
	for _, sig := range authFailureSignatures {
		if strings.Contains(lower, sig) {
			return fmt.Errorf("%w: %s", provider.ErrAuth, diag)
		}
	}
	return fmt.Errorf("antigravity: %w: %s", err, diag)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
