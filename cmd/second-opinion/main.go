// second-opinion sends a document or a diff to a model that did not write
// it, and prints the reviewer's findings. Findings go to stdout; a one-line
// provenance report goes to stderr.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/danhorst/second-opinion/internal/provider"
	"github.com/danhorst/second-opinion/internal/review"
)

const usage = `usage: second-opinion [flags] PATH...
       second-opinion [flags] --diff [BASE]

  PATH...        files to review, concatenated in argument order
  --diff [BASE]  review the git diff against BASE (default: HEAD)
  --provider P   reviewer provider; defaults to $SECOND_OPINION_PROVIDER
  --model M      force a model (provider semantics apply)
  --prompt-file  replace the baked review prompt with a file's contents

subcommands:
  skill install  install the calling-agent skill for claude/codex harnesses

exit codes: 0 review completed; 1 reviewer did not run; 2 usage error
`

type config struct {
	provider   string
	model      string
	promptFile string
	diffMode   bool
	diffBase   string
	paths      []string
	help       bool
}

// parseArgs is a manual loop because --diff takes an optional value, which
// the flag package cannot express.
func parseArgs(args []string) (config, error) {
	cfg := config{diffBase: "HEAD"}
	value := func(i *int, name string) (string, error) {
		*i++
		if *i >= len(args) {
			return "", fmt.Errorf("%s requires a value", name)
		}
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		var err error
		switch arg := args[i]; arg {
		case "-h", "--help":
			cfg.help = true
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
	if cfg.help {
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

	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "second-opinion: %v\n%s", err, usage)
		return 2
	}
	if cfg.help {
		fmt.Fprint(stdout, usage)
		return 0
	}

	name := cfg.provider
	if name == "" {
		name = getenv("SECOND_OPINION_PROVIDER")
	}
	if name == "" {
		fmt.Fprintf(stderr, "second-opinion: no provider selected — pass --provider or set SECOND_OPINION_PROVIDER (registered: %v)\n", review.Providers())
		return 2
	}
	p, err := newProvider(name, cfg.model)
	if err != nil {
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 2
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
		material, err = review.FromDiff(cfg.diffBase)
	} else {
		material, err = review.FromFiles(cfg.paths)
	}
	switch {
	case errors.Is(err, review.ErrNothingToReview):
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "second-opinion: %v\n", err)
		return 2
	}

	res, err := p.Review(ctx, provider.Request{Prompt: prompt, Material: material})
	if err != nil {
		fmt.Fprintf(stderr, "second-opinion: reviewer did not run: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, res.Findings)
	pv := res.Provenance
	fmt.Fprintf(stderr, "reviewed-by: provider=%s model=%s prompt=%s\n", pv.Provider, pv.Model, pv.PromptHash[:12])
	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	args := os.Args[1:]
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
