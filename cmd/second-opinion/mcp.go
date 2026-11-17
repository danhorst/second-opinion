package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/danhorst/second-opinion/internal/mcpserver"
	"github.com/danhorst/second-opinion/internal/review"
	"io"
)

func runMCP(ctx context.Context, args []string, getenv func(string) string, stdin io.ReadCloser, stdout io.WriteCloser, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return runMCPInit(args[1:], getenv, stdout, stderr)
		case "install":
			return runMCPInstall(args[1:], getenv, stdout, stderr)
		case "uninstall":
			return runMCPUninstall(args[1:], getenv, stdout, stderr)
		case "doctor":
			return runMCPDoctor(ctx, args[1:], getenv, stdout, stderr)
		case "connect":
			return runMCPConnect(ctx, args[1:], getenv, stdin, stdout, stderr)
		}
	}
	if len(args) == 0 || args[0] != "serve" {
		fmt.Fprintln(stderr, "usage: second-opinion mcp serve [--transport stdio|http] [--listen 127.0.0.1:8765]")
		fmt.Fprintln(stderr, "       second-opinion mcp init [--token-file PATH]")
		fmt.Fprintln(stderr, "       second-opinion mcp install|uninstall|doctor --harness codex|claude|antigravity [--scope user|project] [--json]")
		fmt.Fprintln(stderr, "       second-opinion mcp connect [--url http://127.0.0.1:8765/mcp] [--token-file PATH]")
		return 2
	}
	fs := flag.NewFlagSet("mcp serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	transport := fs.String("transport", "stdio", "stdio or http")
	listen := fs.String("listen", "127.0.0.1:8765", "literal loopback address for HTTP")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	explicitListen := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "listen" {
			explicitListen = true
		}
	})
	if fs.NArg() != 0 || (*transport != "stdio" && *transport != "http") || (*transport == "stdio" && explicitListen) {
		fmt.Fprintln(stderr, "second-opinion: invalid MCP transport arguments")
		return 2
	}
	server := mcpserver.NewServer(review.Executor{})
	var err error
	if *transport == "stdio" {
		err = mcpserver.ServeStdio(ctx, server, stdin, stdout)
	} else {
		err = mcpserver.ServeHTTP(ctx, server, *listen, getenv("SECOND_OPINION_MCP_TOKEN"))
	}
	if err != nil && ctx.Err() == nil {
		fmt.Fprintln(stderr, "second-opinion: MCP transport failed; check transport configuration, HTTP loopback/token settings, and request limits")
		return 1
	}
	return 0
}
