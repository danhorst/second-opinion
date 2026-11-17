package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/danhorst/second-opinion/internal/harness"
	"github.com/danhorst/second-opinion/internal/mcpcheck"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type doctorReport struct {
	Harness           string           `json:"harness"`
	Path              string           `json:"path"`
	Configuration     string           `json:"configuration"`
	Executable        string           `json:"executable"`
	Connection        string           `json:"connection"`
	Review            string           `json:"review"`
	HarnessAcceptance string           `json:"harness_acceptance"`
	Discovery         *mcpcheck.Result `json:"discovery,omitempty"`
	Message           string           `json:"message"`
}

func runMCPDoctor(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o harness.Options
	fs.StringVar(&o.Harness, "harness", "", "codex, claude, or antigravity (required)")
	fs.StringVar(&o.Scope, "scope", "user", "user or project")
	fs.StringVar(&o.ConfigPath, "config", "", "explicit configuration path")
	asJSON := fs.Bool("json", false, "JSON diagnostic report")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || o.Harness == "" {
		fmt.Fprintln(stderr, "second-opinion: doctor requires --harness and no positional arguments")
		return 2
	}
	path, _, err := harness.Resolve(o, getenv)
	if err != nil {
		fmt.Fprintln(stderr, "second-opinion:", err)
		return 2
	}
	report := doctorReport{Harness: o.Harness, Path: path, Configuration: "failed", Executable: "not_checked", Connection: "not_checked", Review: "not_performed", HarnessAcceptance: "not_verified"}
	finish := func(code int) int {
		var err error
		if *asJSON {
			err = json.NewEncoder(stdout).Encode(report)
		} else {
			_, err = fmt.Fprintf(stdout, "Configuration: %s\nExecutable: %s\nMCP connection: %s\n%s\nReal review: not performed; provider authentication and live harness acceptance: not verified.\n", report.Configuration, report.Executable, report.Connection, report.Message)
		}
		if err != nil {
			return 1
		}
		return code
	}
	registration, exists, err := harness.Inspect(o, getenv)
	if err != nil {
		report.Message = "Registration is invalid or unsupported; preview mcp install --replace to repair it."
		return finish(1)
	}
	if !exists {
		report.Message = "Registration is missing; run mcp install for this harness and scope."
		return finish(1)
	}
	report.Configuration = "valid"
	info, err := os.Stat(registration.Command)
	if err != nil || !info.Mode().IsRegular() {
		report.Executable = "unavailable"
		report.Message = "Configured executable is missing or not a regular file; reinstall with the current executable path."
		return finish(1)
	}
	if _, err = exec.LookPath(registration.Command); err != nil {
		report.Executable = "unavailable"
		report.Message = "Configured executable cannot be executed; check permissions or reinstall."
		return finish(1)
	}
	report.Executable = "available"
	if registration.Args[1] == "connect" {
		if _, err := readMCPTokenFile(registration.Args[5]); err != nil {
			report.Connection = "failed"
			report.Message = "Token file is missing or unsafe; run mcp init with the configured token path."
			return finish(1)
		}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, registration.Command, registration.Args...)
	command.Stderr = io.Discard // Never relay arbitrary transport output or credentials.
	command.WaitDelay = time.Second
	result, err := mcpcheck.Discover(probeCtx, &mcp.CommandTransport{Command: command, TerminateDuration: time.Second})
	if err != nil {
		report.Connection = "failed"
		report.Message = "MCP initialization or discovery failed; check the configured executable and, for HTTP, service status and matching token. No review was run."
		return finish(1)
	}
	report.Connection = "verified"
	report.Discovery = &result
	report.Message = "Initialized Second Opinion and discovered review and list_providers."
	return finish(0)
}
