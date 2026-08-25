// Package codex implements provider.Provider by shelling out to the codex
// CLI. Every invocation suppresses implicit context by construction: an empty
// temp directory, ambient config and rules disabled, project-doc loading
// suppressed, and a read-only ephemeral sandbox. The read-only sandbox does
// not provide strict material-only filesystem isolation.
package codex

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

const providerName = "codex"

// defaultModelMarker is an explicit provenance degradation: current Codex
// JSONL success events do not name the model selected for an unforced run.
const defaultModelMarker = "codex-default"

// chatgptModelRejection is codex's stderr signature for a model the active
// ChatGPT-account auth does not expose. Held in one constant: if OpenAI
// rewords it, forced-model reviews fail loudly with ErrAuth instead of
// silently reviewing on the wrong model.
const chatgptModelRejection = "not supported when using Codex with a ChatGPT account"

// Provider runs reviews through the codex CLI.
type Provider struct {
	model string // forced model; empty means codex's default
}

// Option configures a Provider.
type Option func(*Provider)

// WithModel forces a model. Codex may reject it under ChatGPT-account auth,
// in which case the review is retried once on the default and provenance
// reports what actually ran.
func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

// New returns a codex-backed Provider.
func New(opts ...Option) *Provider {
	p := &Provider{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// buildInvocation constructs the full codex command line and stdin for one
// review. Pure so the cold flags are testable without executing codex.
func buildInvocation(req provider.Request, model, workdir string) (argv []string, stdin string) {
	argv = []string{
		"codex", "exec",
		"-C", workdir,
		"-c", "project_doc_max_bytes=0",
		"--ignore-user-config",
		"--ignore-rules",
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ephemeral",
		"--json",
	}
	if model != "" {
		argv = append(argv, "-m", model)
	}
	argv = append(argv, req.Prompt)
	return argv, req.Material
}

// Review implements provider.Provider.
func (p *Provider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, diag, err := p.run(ctx, req, p.model)
	if err != nil && shouldFallback(ctx, p.model, diag) {
		res, _, err = p.run(ctx, req, "")
	}
	return res, err
}

func shouldFallback(ctx context.Context, model, diag string) bool {
	return ctx.Err() == nil && model != "" && strings.Contains(diag, chatgptModelRejection)
}

// run executes one codex invocation and returns the parsed result. The raw
// diagnostics (JSONL error message plus stderr) are returned alongside so
// Review can apply the fallback predicate.
func (p *Provider) run(ctx context.Context, req provider.Request, model string) (*provider.Result, string, error) {
	workdir, err := os.MkdirTemp("", "second-opinion-codex-")
	if err != nil {
		return nil, "", fmt.Errorf("codex: creating workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	argv, stdin := buildInvocation(req, model, workdir)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = workdir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, stderr.String(), ctxErr
		}
		// Codex reports the real failure as a JSONL error event on stdout;
		// stderr carries only progress chrome.
		diag := errorEventMessage(stdout.Bytes())
		if diag == "" {
			diag = firstLine(stderr.String())
		}
		return nil, diag, classify(err, diag)
	}

	ranModel, findings, hasMessage := parseEvents(stdout.Bytes())
	if !hasMessage {
		return nil, "", fmt.Errorf("codex: successful response contained no recognized agent message")
	}
	if ranModel == "" {
		if model == "" {
			ranModel = defaultModelMarker
		} else {
			// A forced model is the only honest fallback when Codex omits the model
			// event: the command explicitly constrained the session to that model.
			ranModel = model
		}
	}
	return &provider.Result{
		Findings: findings,
		Provenance: provider.Provenance{
			Provider:     providerName,
			Model:        ranModel,
			PromptHash:   provider.HashPrompt(req.Prompt),
			MaterialHash: provider.HashMaterial(req.Material),
		},
	}, stderr.String(), nil
}

// usageLimitSignature marks quota exhaustion on the active codex account —
// the reviewer is unavailable until the limit resets.
const usageLimitSignature = "hit your usage limit"

// classify maps a codex failure onto the contract's did-not-run sentinels.
// diag is the JSONL error-event message when codex emitted one, else the
// first stderr line.
func classify(err error, diag string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: codex binary not found: %v", provider.ErrUnavailable, err)
	}
	lower := strings.ToLower(diag)
	if strings.Contains(lower, usageLimitSignature) {
		return fmt.Errorf("%w: %s", provider.ErrUnavailable, firstLine(diag))
	}
	if strings.Contains(lower, strings.ToLower(chatgptModelRejection)) || strings.Contains(lower, "authentication") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "not logged in") || strings.Contains(lower, "sign in") {
		return fmt.Errorf("%w: %s", provider.ErrAuth, firstLine(diag))
	}
	return fmt.Errorf("codex: %w: %s", err, firstLine(diag))
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
