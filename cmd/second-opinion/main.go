// second-opinion sends a document or a diff to a model that did not write
// it, and prints the reviewer's findings. Findings go to stdout; a one-line
// provenance report goes to stderr.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/danhorst/second-opinion/internal/buildinfo"
	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

// usage renders the registered providers rather than hardcoding them, so
// adding an adapter updates the help text from the registry alone.
func usage() string {
	return fmt.Sprintf(`usage: second-opinion [flags] PATH...
       second-opinion [flags] --diff [BASE]

  PATH...           files to review, concatenated in argument order
  --diff [BASE]     review the git diff against BASE (default: HEAD)
  --provider P      reviewer provider (%s)
                    defaults to $SECOND_OPINION_PROVIDER
  --model M         force a model (provider semantics apply)
  --prompt-file F   replace the baked review prompt with a file's contents
  --json           emit raw findings and full provenance as JSON
  --dry-run        emit the assembled request as JSON; no provider required
  --timeout D      limit the provider call (e.g. 2m); default: unlimited
  --               treat remaining arguments as literal file paths
  --version         print application version and available build identity
  --list-providers  print the registered provider names and exit

subcommands:
  mcp serve      serve stateless review tools over stdio or private HTTP
  skill install  install the calling-agent skill for claude/codex harnesses

exit codes: 0 success (review, preview, or discovery); 1 review/output failure;
            2 usage error
`, strings.Join(review.Providers(), ", "))
}

type config struct {
	provider   string
	model      string
	promptFile string
	diffMode   bool
	diffBase   string
	paths      []string
	help       bool
	listProv   bool
	jsonOutput bool
	dryRun     bool
	timeout    time.Duration
}

// parseArgs is a manual loop because --diff takes an optional value, which
// the flag package cannot express.
func parseArgs(args []string) (config, error) {
	cfg := config{diffBase: "HEAD"}
	value := func(i *int, name string) (string, error) {
		*i++
		if *i >= len(args) || args[*i] == "" || strings.HasPrefix(args[*i], "-") {
			return "", fmt.Errorf("%s requires a value", name)
		}
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		var err error
		switch arg := args[i]; arg {
		case "--":
			cfg.paths = append(cfg.paths, args[i+1:]...)
			i = len(args)
		case "-h", "--help":
			cfg.help = true
		case "--list-providers":
			cfg.listProv = true
		case "--json":
			cfg.jsonOutput = true
		case "--dry-run":
			cfg.dryRun = true
		case "--timeout":
			var duration string
			duration, err = value(&i, arg)
			if err == nil {
				cfg.timeout, err = time.ParseDuration(duration)
				if err != nil || cfg.timeout <= 0 {
					err = fmt.Errorf("--timeout requires a positive duration (e.g. 2m)")
				}
			}
		case "--provider":
			cfg.provider, err = value(&i, arg)
		case "--model":
			cfg.model, err = value(&i, arg)
		case "--prompt-file":
			cfg.promptFile, err = value(&i, arg)
		case "--diff":
			cfg.diffMode = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				cfg.diffBase = args[i]
				if cfg.diffBase == "" {
					err = errors.New("--diff requires a nonempty base when supplied")
				}
			}
		default:
			if strings.HasPrefix(arg, "-") {
				return cfg, fmt.Errorf("unknown flag: %s", arg)
			}
			cfg.paths = append(cfg.paths, arg)
		}
		if err != nil {
			return cfg, err
		}
	}
	if cfg.help || cfg.listProv {
		return cfg, nil
	}
	if cfg.diffMode && len(cfg.paths) > 0 {
		return cfg, errors.New("--diff and PATH arguments are mutually exclusive")
	}
	if !cfg.diffMode && len(cfg.paths) == 0 {
		return cfg, errors.New("nothing to review: give PATH... or --diff")
	}
	return cfg, nil
}

// run is main with its edges injected, so wiring is testable against the
// loopback reference provider.
func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer,
	newProvider func(name, model string) (provider.Provider, error)) int {

	if len(args) == 1 && args[0] == "--version" {
		if _, err := fmt.Fprintln(stdout, buildinfo.Version()); err != nil {
			return 1
		}
		return 0
	}

	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "second-opinion: %v\n%s", err, usage())
		return 2
	}
	if cfg.help {
		fmt.Fprint(stdout, usage())
		return 0
	}
	// Answers a question about the binary, so it runs before target assembly
	// and provider selection: no target, provider, or repo is required.
	if cfg.listProv {
		for _, name := range review.Providers() {
			fmt.Fprintln(stdout, name)
		}
		return 0
	}

	var p provider.Provider
	if !cfg.dryRun {
		name := cfg.provider
		if name == "" {
			name = getenv("SECOND_OPINION_PROVIDER")
		}
		if name == "" {
			fmt.Fprintf(stderr, "second-opinion: no provider selected — pass --provider or set SECOND_OPINION_PROVIDER (registered: %v)\n", review.Providers())
			return 2
		}
		p, err = newProvider(name, cfg.model)
		if err != nil {
			fmt.Fprintf(stderr, "second-opinion: %v\n", err)
			return 2
		}
	}

	prompt := review.BakedPrompt
	if cfg.promptFile != "" {
		b, err := os.ReadFile(cfg.promptFile)
		if err != nil {
			fmt.Fprintf(stderr, "second-opinion: %v\n", err)
			return 2
		}
		prompt = string(b)
	}

	var material string
	if cfg.diffMode {
		material, err = review.FromDiffContext(ctx, cfg.diffBase)
	} else {
		material, err = review.FromFiles(cfg.paths)
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 1
	case errors.Is(err, review.ErrNothingToReview):
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 2
	}

	req := provider.Request{Prompt: prompt, Material: material}
	if cfg.dryRun {
		if !utf8.ValidString(req.Prompt) || !utf8.ValidString(req.Material) {
			fmt.Fprintln(stderr, "second-opinion: JSON preview requires UTF-8 prompt and material")
			return 2
		}
		return writeJSON(stdout, stderr, requestOutput{
			SchemaVersion: 1, Kind: "request", Prompt: req.Prompt,
			Material: req.Material, PromptHash: provider.HashPrompt(req.Prompt),
		})
	}
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}
	res, err := p.Review(ctx, req)
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil {
		fmt.Fprintf(stderr, "second-opinion: reviewer did not run: %v\n", err)
		return 1
	}
	if res == nil {
		fmt.Fprintln(stderr, "second-opinion: provider returned no result")
		return 1
	}
	pv := res.Provenance
	if cfg.jsonOutput {
		if !utf8.ValidString(res.Findings) || !utf8.ValidString(pv.Provider) || !utf8.ValidString(pv.Model) || !utf8.ValidString(pv.PromptHash) {
			fmt.Fprintln(stderr, "second-opinion: JSON output requires UTF-8 findings and provenance")
			return 1
		}
		if code := writeJSON(stdout, stderr, reviewOutput{
			SchemaVersion: 1, Kind: "review", Raw: res.Findings,
			Provenance: provenanceOutput{pv.Provider, pv.Model, pv.PromptHash},
		}); code != 0 {
			return code
		}
	} else if _, err := fmt.Fprintln(stdout, res.Findings); err != nil {
		fmt.Fprintf(stderr, "second-opinion: writing output: %v\n", err)
		return 1
	}
	if _, err := fmt.Fprintf(stderr, "reviewed-by: provider=%s model=%s prompt=%.12s\n", pv.Provider, pv.Model, pv.PromptHash); err != nil {
		return 1
	}
	return 0
}

// These versioned CLI wire types are independent of provider internals and
// the future best-effort findings parser. Raw remains uninterpreted text.
type provenanceOutput struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	PromptHash string `json:"prompt_hash"`
}

type reviewOutput struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	Raw           string           `json:"raw"`
	Provenance    provenanceOutput `json:"provenance"`
}

type requestOutput struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	Prompt        string `json:"prompt"`
	Material      string `json:"material"`
	PromptHash    string `json:"prompt_hash"`
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintf(stderr, "second-opinion: writing output: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "mcp" {
		os.Exit(runMCP(ctx, args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr))
	}
	if len(args) > 0 && args[0] == "skill" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "second-opinion: %v\n", err)
			os.Exit(1)
		}
		deps := skillDeps{
			home:     home,
			lookPath: exec.LookPath,
			runInstall: func(pluginDir string) (string, error) {
				out, err := exec.CommandContext(ctx, "agy", "plugin", "install", pluginDir).CombinedOutput()
				return string(out), err
			},
		}
		os.Exit(runSkill(args[1:], deps, os.Stdout, os.Stderr))
	}
	os.Exit(run(ctx, args, os.Getenv, os.Stdout, os.Stderr, review.NewProvider))
}
