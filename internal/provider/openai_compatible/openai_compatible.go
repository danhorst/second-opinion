// Package openai_compatible implements provider.Provider over the common
// non-streaming OpenAI-compatible chat-completions API.
package openai_compatible

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/danhorst/second-opinion/internal/provider"
)

const (
	providerName         = "openai-compatible"
	defaultBaseURL       = "https://api.openai.com/v1"
	unknownModelMarker   = "openai-compatible-model-unknown"
	maxDiagnosticBytes   = 4096
	defaultResponseBytes = 8 << 20
)

// Provider reviews through an OpenAI-compatible HTTP endpoint.
type Provider struct {
	captureLimits provider.CaptureLimits
	baseURL       string
	apiKey        string
	model         string
	client        *http.Client
}

// Option configures a Provider.
type Option func(*Provider)

// WithBaseURL overrides the endpoint base URL.
func WithBaseURL(baseURL string) Option {
	return func(p *Provider) { p.baseURL = baseURL }
}

// WithAPIKey overrides the bearer token.
func WithAPIKey(apiKey string) Option {
	return func(p *Provider) { p.apiKey = apiKey }
}

// WithModel overrides the configured model.
func WithModel(model string) Option {
	return func(p *Provider) { p.model = model }
}

// WithCaptureLimits bounds the HTTP response: StdoutBytes for success and
// StderrBytes for an error response. Zero uses the adapter's existing defaults.
func WithCaptureLimits(limits provider.CaptureLimits) Option {
	return func(p *Provider) { p.captureLimits = limits }
}

// WithHTTPClient injects an HTTP client, primarily for transport tests.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) { p.client = client }
}

// New returns a provider configured from the environment and options.
// SECOND_OPINION_API_BASE_URL defaults to the OpenAI API; the model option
// takes precedence over SECOND_OPINION_API_MODEL.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL: strings.TrimRight(envOrDefault("SECOND_OPINION_API_BASE_URL", defaultBaseURL), "/"),
		apiKey:  env("SECOND_OPINION_API_KEY"),
		model:   env("SECOND_OPINION_API_MODEL"),
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Isolation certifies only this client's mechanics. The configured endpoint,
// credentials, and HTTP transport are trusted, not sandboxed by this adapter.
func (p *Provider) Isolation() provider.Isolation { return provider.IsolationLocalMaterialOnly }

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content      string            `json:"content"`
			ToolCalls    []json.RawMessage `json:"tool_calls"`
			FunctionCall json.RawMessage   `json:"function_call"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Review implements provider.Provider.
func (p *Provider) Review(ctx context.Context, req provider.Request) (*provider.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.apiKey == "" {
		return nil, errors.New("openai-compatible: SECOND_OPINION_API_KEY is required")
	}
	if p.model == "" {
		return nil, errors.New("openai-compatible: model is required via --model or SECOND_OPINION_API_MODEL")
	}
	if p.baseURL == "" {
		return nil, errors.New("openai-compatible: SECOND_OPINION_API_BASE_URL is required")
	}

	body, err := json.Marshal(chatRequest{
		Model: p.model,
		Messages: []message{
			{Role: "system", Content: req.Prompt},
			{Role: "user", Content: req.Material},
		},
		Stream: false,
	})
	if err != nil {
		return nil, fmt.Errorf("openai-compatible: encoding request: %w", err)
	}

	endpoint := strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai-compatible: creating request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	// Do not mutate a shared/injected client, and never let a response choose a
	// new destination. Custom transports remain trusted execution dependencies.
	boundedClient := *client
	boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := boundedClient.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: openai-compatible request: %v", provider.ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, errors.New("openai-compatible: redirects are disabled; configure the final endpoint URL")
	}

	success := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	budget := defaultResponseBytes
	configured := p.captureLimits.StdoutBytes
	if !success {
		budget = maxDiagnosticBytes
		configured = p.captureLimits.StderrBytes
	}
	if configured > 0 {
		budget = configured
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, int64(budget)+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("openai-compatible: reading response: %w", err)
	}
	if len(responseBody) > budget {
		if success || configured > 0 {
			return nil, provider.ErrOutputLimit
		}
		responseBody = responseBody[:budget] // bounded legacy error diagnostics
	}
	if !success {
		return nil, classifyStatus(resp.StatusCode, responseBody)
	}

	var decoded chatResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("openai-compatible: malformed response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, errors.New("openai-compatible: response contained no choices")
	}
	for _, choice := range decoded.Choices {
		call := bytes.TrimSpace(choice.Message.FunctionCall)
		if len(choice.Message.ToolCalls) > 0 || (len(call) > 0 && !bytes.Equal(call, []byte("null"))) || choice.FinishReason == "tool_calls" || choice.FinishReason == "function_call" {
			return nil, errors.New("openai-compatible: reviewer requested tools; material-only reviews cannot execute tools")
		}
	}

	model := decoded.Model
	if model == "" {
		model = unknownModelMarker
	}
	return &provider.Result{
		Findings: decoded.Choices[0].Message.Content,
		Provenance: provider.Provenance{
			Provider:     providerName,
			Model:        model,
			PromptHash:   provider.HashPrompt(req.Prompt),
			MaterialHash: provider.HashMaterial(req.Material),
		},
	}, nil
}

func classifyStatus(status int, body []byte) error {
	detail := firstLine(string(body))
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: openai-compatible HTTP %d: %s", provider.ErrAuth, status, detail)
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return fmt.Errorf("%w: openai-compatible HTTP %d: %s", provider.ErrUnavailable, status, detail)
	default:
		if status >= 500 {
			return fmt.Errorf("%w: openai-compatible HTTP %d: %s", provider.ErrUnavailable, status, detail)
		}
		return fmt.Errorf("openai-compatible HTTP %d: %s", status, detail)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func env(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}

func envOrDefault(name, fallback string) string {
	if value := env(name); value != "" {
		return value
	}
	return fallback
}
