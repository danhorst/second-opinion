# Changelog

## [Unreleased]

## [0.1.0] - 2026-07-14

### Added

- `second-opinion` CLI: review files or a git diff (`--diff [BASE]`, default `HEAD`), select a provider (`--provider`, or `$SECOND_OPINION_PROVIDER`), force a model (`--model`), override the baked prompt (`--prompt-file`). Findings on stdout, one-line provenance on stderr.
- `claude` provider: shells out to `claude -p` with tools and setting sources disabled, so the reviewer can't read the repo or inherit user memory. Provenance names the model that actually ran.
- `antigravity` provider: shells out to `agy --print` from an empty temp directory. Provenance names the forced model, or `antigravity-default` when none is set.
- `second-opinion skill install [--harness claude|codex|all]`: installs the calling-agent triage skill for Claude Code and Codex, and as an Antigravity plugin via `agy plugin install`.
- `codex` provider (**beta, untested against the real binary**): shells out to `codex exec` with project-doc loading suppressed and a read-only ephemeral sandbox. Unit tests and the mocked conformance suite pass; real-binary conformance is blocked on a codex account usage limit and hasn't been verified yet.
- `README.md` and MIT-0 `LICENSE`.
