// Package claude implements provider.Provider by shelling out to the claude
// CLI. The coldest posture of the adapters: empty temp-directory cwd, all
// tools disabled (repo reads are impossible, not merely unprompted), and all
// setting sources suppressed (no user or project instruction and memory
// files). Material travels by value inside the composed instruction —
// claude's print mode with a prompt argument does not read stdin.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/danhorst/second-opinion/internal/provider"
)

const providerName = "claude"

// authFailureSignatures classify a did-not-run as ErrAuth.
var authFailureSignatures = []string{
	"log in",
	"login",
	"authentication",
	"invalid api key",
	"unauthorized",
}

// usageLimitSignature marks quota exhaustion — the reviewer is unavailable
// until the limit resets.
const usageLimitSignature = "usage limit"

// Provider runs reviews through the claude CLI.
type Provider struct {
	model string // forced model (alias or full ID); empty means claude's default
}

// Option configures a Provider.
type Option func(*Provider)

// WithModel forces a model, passed to claude as --model.
func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

// New returns a claude-backed Provider.
func New(opts ...Option) *Provider {
	p := &Provider{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// composeInstruction merges the prompt and material into the single
// instruction claude receives. Pure and deterministic.
func composeInstruction(req provider.Request) string {
	return req.Prompt +
		"\n\n--- MATERIAL (verbatim, assembled by the caller) ---\n" +
		req.Material +
		"\n--- END MATERIAL ---\n"
}

// buildInvocation constructs the claude command line. Pure so the cold
// flags are testable without executing claude. Isolation comes from the
// caller running the process with its working directory set to an empty
// temp directory; there is no stdin.
func buildInvocation(req provider.Request, model string) (argv []string) {
	argv = []string{
		"claude", "-p",
		"--output-format", "json",
		"--tools", "",
		"--setting-sources", "",
	}
	if model != "" {
		argv = append(argv, "--model", model)
	}
	argv = append(argv, composeInstruction(req))
	return argv
}

// envelope is claude -p --output-format json's result shape, reduced to the
// fields the adapter reads. Captured 2026-07-10 from claude CLI 2.1.206.
type envelope struct {
	Type       string                `json:"type"`
	Subtype    string                `json:"subtype"`
	IsError    bool                  `json:"is_error"`
	Result     string                `json:"result"`
	ModelUsage map[string]modelUsage `json:"modelUsage"`
}

type modelUsage struct {
	OutputTokens int `json:"outputTokens"`
}

// dominantModel returns the modelUsage entry with the most output tokens —
// the model that wrote the findings. Harness helper models contribute
// negligible output.
func (e envelope) dominantModel() string {
	best, bestTokens := "", -1
	for id, usage := range e.ModelUsage {
		if usage.OutputTokens > bestTokens || (usage.OutputTokens == bestTokens && id < best) {
			best, bestTokens = id, usage.OutputTokens
		}
	}
	return best
}

// Review implements provider.Provider.
func (p *Provider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	workdir, err := os.MkdirTemp("", "second-opinion-claude-")
	if err != nil {
		return nil, fmt.Errorf("claude: creating workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	argv := buildInvocation(req, p.model)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = workdir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	var env envelope
	parsed := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &env) == nil

	if runErr != nil || (parsed && env.IsError) {
		diag := ""
		if parsed {
			diag = firstLine(env.Result)
		}
		if diag == "" {
			diag = firstLine(stderr.String())
		}
		if runErr == nil {
			runErr = fmt.Errorf("result envelope reports error (%s)", env.Subtype)
		}
		return nil, classify(runErr, diag)
	}
	if !parsed {
		return nil, fmt.Errorf("claude: unparseable result envelope: %s", firstLine(stdout.String()))
	}

	model := env.dominantModel()
	if model == "" {
		model = p.model
		if model == "" {
			model = "claude-default"
		}
	}
	return &provider.Result{
		Findings: strings.TrimSpace(env.Result),
		Provenance: provider.Provenance{
			Provider:   providerName,
			Model:      model,
			PromptHash: provider.HashPrompt(req.Prompt),
		},
	}, nil
}

// classify maps a claude failure onto the contract's did-not-run sentinels.
func classify(err error, diag string) error {
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: claude binary not found: %v", provider.ErrUnavailable, err)
	}
	lower := strings.ToLower(diag)
	if strings.Contains(lower, usageLimitSignature) {
		return fmt.Errorf("%w: %s", provider.ErrUnavailable, diag)
	}
	for _, sig := range authFailureSignatures {
		if strings.Contains(lower, sig) {
			return fmt.Errorf("%w: %s", provider.ErrAuth, diag)
		}
	}
	return fmt.Errorf("claude: %w: %s", err, diag)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
