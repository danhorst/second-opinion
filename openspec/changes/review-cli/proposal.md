# review-cli

## Why

Three conformant adapters exist but nothing a human or agent can invoke.
The CLI turns the engine into a usable tool and retires `dotfiles/claude/bin/second-opinion.sh` — which still runs its reviewer inside the caller's repo, so every review it performs today is potentially contaminated by the very context the prompt tells the reviewer to ignore.

## What Changes

- New binary `cmd/second-opinion`: review files (`PATH...`) or the working diff (`--diff [BASE]`, BASE defaulting to HEAD).
- New package `internal/review`: target assembly (file concatenation, `git diff`) and the baked adversarial prompt, migrated from the predecessor script and made transport-neutral (the old prompt referenced codex's `<stdin>` block by name).
- Provider selection: `--provider` flag, defaulting from `SECOND_OPINION_PROVIDER`; with neither, a usage error listing the registered providers — the unbiased tool ships no baked provider preference.
- `--model` passthrough to the selected provider; `--prompt-file` to override the baked prompt.
- Findings to stdout; a one-line provenance report to stderr, so piped output stays clean findings while what-reviewed-this remains visible.
- Exit codes: 0 = review completed (findings are data, not errors), 1 = reviewer did not run, 2 = usage error.

## Capabilities

### New Capabilities
- `review-prompt`: the baked adversarial prompt as an engine asset — transport-neutral wording, override mechanics, and its identity in provenance.
- `review-cli`: the command-line front-end — target assembly, provider selection, output and exit-code contract.

### Modified Capabilities
None.

## Non-goals

- The MCP front-end (next change; it reuses `internal/review` and the provider registry).
- Triage — findings pass through raw; clustering and ranking stay with the calling agent, per the open question.
- Editing the dotfiles repo — retiring `second-opinion.sh` and updating the skill doc happen there, manually, once this binary is installed.
- Release wiring (`bootstrap-release` follows this change, now that there is a binary to ship).
- Same-model enforcement — the CLI surfaces provenance; it does not yet compare it to an author declaration.

## Impact

- New `cmd/second-opinion` and `internal/review`; a small provider registry (name → constructor) shared by future front-ends.
- First user-facing surface: the flag names and exit codes chosen here become compatibility constraints.
- No changes to the provider contract or any adapter.
