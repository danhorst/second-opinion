package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/danhorst/second-opinion/internal/harness"
	"io"
)

func runMCPInstall(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	return runMCPSetup(args, getenv, stdout, stderr, false)
}
func runMCPUninstall(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	return runMCPSetup(args, getenv, stdout, stderr, true)
}
func runMCPSetup(args []string, getenv func(string) string, stdout, stderr io.Writer, remove bool) int {
	var o harness.Options
	var asJSON bool
	fs := flag.NewFlagSet("mcp install/uninstall", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.Harness, "harness", "", "codex, claude, or antigravity (required)")
	fs.StringVar(&o.Scope, "scope", "user", "user or project")
	fs.StringVar(&o.ConfigPath, "config", "", "explicit harness configuration path")
	fs.StringVar(&o.Executable, "executable", "", "absolute Second Opinion executable override")
	fs.StringVar(&o.Transport, "transport", "stdio", "stdio or http")
	fs.StringVar(&o.URL, "url", "", "loopback HTTP MCP endpoint")
	fs.StringVar(&o.TokenFile, "token-file", "", "absolute private service token file")
	fs.BoolVar(&o.Replace, "replace", false, "replace conflicting registration")
	fs.BoolVar(&o.DryRun, "dry-run", false, "preview without writing")
	fs.BoolVar(&asJSON, "json", false, "JSON report")
	if e := fs.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || o.Harness == "" {
		fmt.Fprintln(stderr, "second-opinion: --harness is required and positional arguments are unsupported")
		return 2
	}
	_, desired, resolveErr := harness.Resolve(o, getenv)
	if resolveErr != nil {
		fmt.Fprintln(stderr, "second-opinion:", resolveErr)
		return 2
	}
	if !remove && len(desired.Args) == 6 {
		if e := validateMCPTokenFile(desired.Args[5]); e != nil {
			fmt.Fprintln(stderr, "second-opinion:", e)
			return 1
		}
	}
	r, e := harness.Apply(o, getenv, remove)
	if e != nil {
		fmt.Fprintln(stderr, "second-opinion:", e)
		return 1
	}
	if asJSON {
		if json.NewEncoder(stdout).Encode(r) != nil {
			return 1
		}
	} else {
		if o.DryRun {
			entry, _ := json.MarshalIndent(r.Entry, "", "  ")
			fmt.Fprintln(stdout, string(entry))
		}
		if _, e = fmt.Fprintf(stdout, "%s %s: changed=%t dry_run=%t\n", r.Operation, r.Path, r.Changed, r.DryRun); e != nil {
			return 1
		}
	}
	return 0
}
