package mcpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/danhorst/second-opinion/internal/buildinfo"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOnlyToolsCapabilityAndNoSubscriptions(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			server := NewServer(review.Executor{})
			failures := make(chan error, 1)
			server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
				return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
					result, err := next(ctx, method, req)
					if method == "subscriptions/listen" || method == "resources/subscribe" {
						failures <- err
					}
					return result, err
				}
			})
			st, ct := mcp.NewInMemoryTransports()
			ss, err := server.Connect(t.Context(), st, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer ss.Close()
			client, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), ct, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if got := client.InitializeResult().ServerInfo.Version; got != buildinfo.Version() {
				t.Fatalf("server version %s", got)
			}
			caps := client.InitializeResult().Capabilities
			if caps == nil || caps.Tools == nil || caps.Tools.ListChanged || caps.Logging != nil || caps.Resources != nil || caps.Prompts != nil || caps.Completions != nil {
				t.Fatalf("unexpected capabilities: %+v", caps)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			_ = client.Subscribe(ctx, &mcp.SubscribeParams{URI: "file:///should-never-be-read"})
			select {
			case err = <-failures:
			case <-ctx.Done():
				t.Fatal("subscription handler did not reject promptly")
			}
			var rpcErr *jsonrpc.Error
			if !errors.As(err, &rpcErr) || rpcErr.Code != jsonrpc.CodeMethodNotFound {
				t.Fatalf("subscription got %v, want method-not-found", err)
			}
			if ctx.Err() != nil {
				t.Fatal("subscription blocked instead of rejection")
			}
		})
	}
}
