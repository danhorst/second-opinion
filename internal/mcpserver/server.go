package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/danhorst/second-opinion/internal/buildinfo"
	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer owns only bounded active executions, never review history.
func NewServer(executor review.Executor) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "second-opinion", Version: buildinfo.Version()}, &mcp.ServerOptions{
		Capabilities:              &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
		SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25"},
		Logger:                    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	// The SDK implements subscription methods even when no subscriptions are
	// advertised. Reject them explicitly so no protocol call can park a stream.
	s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "subscriptions/listen", "resources/subscribe", "resources/unsubscribe":
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeMethodNotFound, Message: "Subscriptions are not supported"}
			}
			return next(ctx, method, req)
		}
	})
	no, yes := false, true
	slots := make(chan struct{}, MaxConcurrentReviews)
	s.AddTool(&mcp.Tool{Name: "list_providers", Description: "List registered reviewers and fixed limits without probing authentication or spending a model call.", InputSchema: EmptyInputSchema, OutputSchema: ProvidersOutputSchema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &no, OpenWorldHint: &no}}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := ValidateEmptyInput(req.Params.Arguments); err != nil {
			return toolFailure(*err), nil
		}
		return toolResult(ProvidersOutput{review.ProviderIsolation(), 1, "providers", review.Providers(), EffectiveLimits()}, false), nil
	})
	s.AddTool(&mcp.Tool{Name: "review", Description: "Review complete material supplied by value. Material may leave this machine and incur cost. Select a model different from the author; default model markers do not establish independence. No workspace reads, stored results, or automatic retries.", InputSchema: ReviewInputSchema, OutputSchema: ReviewOutputSchema, Annotations: &mcp.ToolAnnotations{DestructiveHint: &no, OpenWorldHint: &yes}}, func(ctx context.Context, req *mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		defer func() {
			if recover() != nil {
				result = executionFailure(errors.New("provider panic"))
				err = nil
			}
		}()
		in, invalid := ParseReviewInput(req.Params.Arguments)
		if invalid != nil {
			return toolFailure(*invalid), nil
		}
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			return toolFailure(ToolError{"busy", "Both review slots are occupied; no review was started.", "not_started"}), nil
		}
		ctx, cancel := context.WithTimeout(ctx, time.Duration(in.TimeoutMS)*time.Millisecond)
		defer cancel()
		value, callErr := executor.Review(ctx, review.ExecutionRequest{Provider: in.Provider, Model: in.Model, RequireIsolation: in.RequireIsolation, Material: in.Material, Prompt: in.Prompt, CaptureLimits: provider.CaptureLimits{StdoutBytes: MaxStdoutBytes, StderrBytes: MaxStderrBytes}})
		if callErr != nil {
			return executionFailure(callErr), nil
		}
		if ctx.Err() != nil {
			return executionFailure(ctx.Err()), nil
		}
		if len(value.Findings) > MaxRawBytes {
			return executionFailure(provider.ErrOutputLimit), nil
		}
		return toolResult(ReviewOutput{value.Isolation, 1, "review", value.Findings, Provenance{value.Provenance.Provider, value.Provenance.Model, value.Provenance.PromptHash, value.Provenance.MaterialHash}, Receipt{provider.HashMaterial(in.Material), len(in.Material), in.TimeoutMS}}, false), nil
	})
	return s
}

func toolResult(value any, failed bool) *mcp.CallToolResult {
	encoded, err := json.Marshal(value)
	if err != nil {
		return executionFailure(errors.New("encoding failed"))
	}
	result := &mcp.CallToolResult{StructuredContent: value, Content: []mcp.Content{&mcp.TextContent{Text: string(encoded)}}, IsError: failed}
	// Reserve framing space; transports also enforce the complete message cap.
	full, err := json.Marshal(result)
	if err != nil || len(full) > MaxResponseBytes-4096 {
		return executionFailure(provider.ErrOutputLimit)
	}
	return result
}
func toolFailure(err ToolError) *mcp.CallToolResult {
	return toolResult(ErrorOutput{1, "error", err}, true)
}
func executionFailure(err error) *mcp.CallToolResult {
	code, message := "provider_failed", "Reviewer failed or returned an invalid result; completion is unknown."
	switch {
	case errors.Is(err, provider.ErrIsolationUnsupported):
		return toolFailure(ToolError{"isolation_unsupported", "Provider cannot enforce the requested local material boundary; no review was started.", "not_started"})
	case errors.Is(err, provider.ErrOutputLimit):
		code, message = "output_too_large", "Reviewer output exceeded a configured limit; completion is unknown."
	case errors.Is(err, context.DeadlineExceeded):
		code, message = "timeout", "Review exceeded its deadline; completion is unknown."
	case errors.Is(err, context.Canceled):
		code, message = "cancelled", "Review was cancelled; completion is unknown."
	case errors.Is(err, provider.ErrUnavailable):
		code, message = "unavailable", "Reviewer is unavailable; check installation and quota. Completion is unknown."
	case errors.Is(err, provider.ErrAuth):
		code, message = "auth", "Reviewer authentication failed; check provider credentials. Completion is unknown."
	}
	return toolFailure(ToolError{code, message, "unknown"})
}
