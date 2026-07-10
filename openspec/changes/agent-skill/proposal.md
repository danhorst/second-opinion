# agent-skill

## Why

The tool's value split is deliberate — the binary finds, the calling agent triages — but the triage discipline lives in one machine's dotfiles, welded to the retired script.
A skill shipped inside the binary, installable per harness, makes the calling-agent half of the tool travel with the tool.

## What Changes

- Canonical skill at `skills/second-opinion/SKILL.md`: invocation, exit-code interpretation, provenance reading, reviewer-≠-author selection, and the five-step triage discipline — provider-neutral throughout.
- The skill is embedded in the binary (`go:embed`) so `brew install` carries it.
- New subcommand: `second-opinion skill install [--harness claude|codex|all] [--stdout]` — Claude Code gets the SKILL.md verbatim, Codex gets a body-only custom prompt, and Antigravity is reached through its Claude-format import (`agy plugin import claude`), run automatically when `agy` is on PATH.
- Installs are idempotent overwrites of files the tool owns, stamped as generated.
- Roster correction: Google has deprecated the Gemini CLI in favor of the Antigravity toolchain, so the planned `gemini` adapter is removed from the milestones and project context — the antigravity adapter already carries Google.

## Capabilities

### New Capabilities
- `agent-skill`: the calling-agent skill — canonical content requirements, per-harness installation, generated-file semantics.

### Modified Capabilities
None — the review contract and CLI review behavior are untouched; `skill` is a new subcommand beside them.

## Non-goals

- Deleting the dotfiles command and script (manual follow-up in the dotfiles repo).
- An MCP-facing tool description (the MCP front-end change owns that).
- Auto-install at brew time — installation stays an explicit user action, like `openspec init`.
- A gemini install target or adapter — the CLI is deprecated; the `GEMINI.md` conformance canary stays, since instruction files outlive the CLIs that introduced them.

## Impact

- New `skills/` directory, `cmd/second-opinion/skill.go`, a dispatch line in `main.go`.
- `AGENTS.md` and `openspec/config.yaml` lose the planned gemini adapter.
- Once installed, both halves of the old dotfiles `/second-opinion` are superseded.
