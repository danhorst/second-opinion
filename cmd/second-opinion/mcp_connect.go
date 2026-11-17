package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/danhorst/second-opinion/internal/mcpclient"
)

func runMCPConnect(ctx context.Context, args []string, getenv func(string) string, stdin io.ReadCloser, stdout io.WriteCloser, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp connect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	endpoint := fs.String("url", "http://127.0.0.1:8765/mcp", "local MCP URL")
	tokenPath := fs.String("token-file", "", "private token file")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		return 2
	}
	if err := mcpclient.ValidateURL(*endpoint); err != nil {
		fmt.Fprintln(stderr, "second-opinion: invalid local MCP URL")
		return 2
	}
	if *tokenPath == "" {
		*tokenPath = getenv("SECOND_OPINION_MCP_TOKEN_FILE")
	}
	if *tokenPath == "" {
		if getenv("HOME") == "" {
			fmt.Fprintln(stderr, "second-opinion: token path required")
			return 1
		}
		*tokenPath = filepath.Join(getenv("HOME"), ".config", "second-opinion", "mcp-token")
	}
	token, err := readMCPTokenFile(*tokenPath)
	if err != nil {
		fmt.Fprintln(stderr, "second-opinion:", err)
		return 1
	}
	if err := mcpclient.Bridge(ctx, *endpoint, token, stdin, stdout); err != nil {
		fmt.Fprintln(stderr, "second-opinion: MCP connection failed")
		return 1
	}
	return 0
}
