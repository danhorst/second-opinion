# Changelog

## [Unreleased]

### Added

- Shared granular application version in `--version` and MCP discovery, with Git build identity and release-stamped source archives.

- Stateless MCP gateway: `mcp serve` over stdio or authenticated loopback HTTP, with explicit material/provider inputs, versioned raw/provenance outcomes, safe errors, cancellation, and fixed input/output/concurrency budgets.
- Optional provider capture limits cancel overflowing subprocesses without returning partial findings; gateway tokens are excluded from child environments.

- Harness controls: `--json` for raw findings with full provenance, `--dry-run` for provider-free request inspection, `--timeout` for a shared review/fallback deadline, and `--` for literal paths.
- Assembly benchmarks and pull-request CI for vet, tests, race checks, and builds.

### Fixed

- Provider transport validation rejects incomplete/malformed Codex events and non-result Claude JSON; explicit empty results remain valid.
- Provider cancellation terminates invocation process groups on macOS/Linux/BSD and bounds inherited-pipe waits; OS argument-limit failures include actionable guidance.
- Stronger cold-review controls for Codex and Claude, plus a positive-control-gated absolute-path conformance check. Codex and Antigravity pass; Claude verification is blocked by expired OAuth.
- Claude request material travels on stdin, avoiding argument-size limits; expired OAuth is classified as an authentication error.
- Codex skill installation writes a native skill under `~/.agents/skills`, preserving legacy prompt files.
- Diff assembly disables color, external diff, and textconv, separates revisions from paths, and honors cancellation.
- Required flag values cannot silently consume another flag; output write failures return nonzero.
- File assembly avoids a material-sized temporary string when checking for whitespace-only input.
- README and calling-agent instructions distinguish recorded conformance from unverified model identities and document untracked-file exclusions.

## [0.1.0] - 2026-07-14

### Added

- `second-opinion` CLI: review files or a git diff (`--diff [BASE]`, default `HEAD`), select a provider (`--provider`, or `$SECOND_OPINION_PROVIDER`), force a model (`--model`), override the baked prompt (`--prompt-file`). Findings on stdout, one-line provenance on stderr.
- `claude` provider: shells out to `claude -p` with tools and setting sources disabled, so the reviewer can't read the repo or inherit user memory. Provenance names the model that actually ran.
- `antigravity` provider: shells out to `agy --print` from an empty temp directory. Provenance names the forced model, or `antigravity-default` when none is set.
- `second-opinion skill install [--harness claude|codex|all]`: installs the calling-agent triage skill for Claude Code and Codex, and as an Antigravity plugin via `agy plugin install`.
- `codex` provider: shells out to `codex exec` with project-doc loading suppressed and a read-only ephemeral sandbox. Passes the full conformance suite against the real binary. Provenance names the forced model, or `codex-default` when none is set — codex's `--json` stream carries no model field to extract.
- `README.md` and MIT-0 `LICENSE`.
