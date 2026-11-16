package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error { return nil }

func TestMCPArguments(t *testing.T) {
	for _, tt := range []struct {
		args []string
		code int
	}{
		{nil, 2}, {[]string{"nope"}, 2}, {[]string{"serve", "--help"}, 0},
		{[]string{"serve", "--transport", "bad"}, 2},
		{[]string{"serve", "--listen", "127.0.0.1:8765"}, 2},
		{[]string{"serve", "extra"}, 2}, {[]string{"serve", "--transport"}, 2},
		{[]string{"serve", "--transport", "http"}, 1},
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			got := runMCP(context.Background(), tt.args, func(string) string { return "" }, io.NopCloser(strings.NewReader("")), writeCloser{&out}, &diagnostic)
			if got != tt.code {
				t.Fatalf("exit=%d want=%d diagnostic=%s", got, tt.code, diagnostic.String())
			}
			if out.Len() != 0 {
				t.Fatal("startup diagnostics reached protocol stdout")
			}
		})
	}
}

func TestMCPProcessHelper(t *testing.T) {
	if os.Getenv("SECOND_OPINION_TEST_MCP_PROCESS") != "1" {
		return
	}
	os.Exit(runMCP(context.Background(), []string{"serve"}, os.Getenv, os.Stdin, os.Stdout, os.Stderr))
}

func TestMCPProcessStdio(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPProcessHelper$")
			cmd.Env = append(os.Environ(), "SECOND_OPINION_TEST_MCP_PROCESS=1")
			var diagnostic bytes.Buffer
			cmd.Stderr = &diagnostic
			client := mcp.NewClient(&mcp.Implementation{Name: "process-test", Version: "1"}, nil)
			session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, &mcp.ClientSessionOptions{ProtocolVersion: version})
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_providers", Arguments: map[string]any{}})
			if err != nil || result.IsError {
				t.Fatalf("discovery: %v %v", result, err)
			}
			if err := session.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if diagnostic.Len() != 0 {
				t.Fatalf("unexpected diagnostics: %s", diagnostic.String())
			}
		})
	}
}
