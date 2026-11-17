package mcpcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/danhorst/second-opinion/internal/mcpserver"
	"github.com/danhorst/second-opinion/internal/review"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDiscoveryAgainstRealServer(t *testing.T) {
	st, ct := mcp.NewInMemoryTransports()
	ss, err := mcpserver.NewServer(review.Executor{}).Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	result, err := Discover(t.Context(), ct)
	if err != nil {
		t.Fatal(err)
	}
	if result.ServerName != "second-opinion" || result.ServerVersion == "" || len(result.Tools) != 2 {
		t.Fatalf("unexpected discovery: %+v", result)
	}
}

func TestDiscoveryRejectsMissingTools(t *testing.T) {
	st, ct := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "second-opinion", Version: "test"}, nil)
	ss, err := server.Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	_, err = Discover(t.Context(), ct)
	if err == nil || !strings.Contains(err.Error(), "required tools") {
		t.Fatalf("got %v", err)
	}
}

func TestDiscoveryCancellation(t *testing.T) {
	_, ct := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Discover(ctx, ct); err == nil {
		t.Fatal("cancelled discovery succeeded")
	}
}
