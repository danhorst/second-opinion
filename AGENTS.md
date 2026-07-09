Adversarial review: send a document or a diff to a model that did not write it, and get findings back.

Purpose, design guarantees, and constraints live in `openspec/config.yaml` under `context:` — read it first.
Specs in `openspec/specs/` are the current truth; proposals live in `openspec/changes/`.
Drive the work with the OpenSpec skills — propose, apply, then sync or archive.

## Critical constraints

**The reviewer runs cold: nothing implicit.**
Everything the reviewer sees was explicitly assembled into the material — an author may bundle supporting files deliberately, but a provider that slurps context on its own violates the guarantee.
This is easy to violate by accident: `codex` auto-loads `AGENTS.md` from its working directory, so an adapter that merely runs `codex exec` in the caller's repo hands the reviewer the very decisions the prompt tells it not to restate.

**The reviewer finds; it does not rank.**
Novelty is the axis it cannot judge.
Where that judgment lives is an open design question — do not answer it in code without a spec.

## Testing

Cover behavior, and name what you cannot cover.

- Unit-test the pure logic: argument parsing, target assembly, prompt selection, the auth-fallback predicate.
- Each provider adapter passes a shared conformance suite, run against the real binary or endpoint behind a build tag.
- Do not mock the provider boundary to reach a coverage number.
- Anything genuinely untestable is named in this file. An untested path nobody wrote down is a bug waiting for a user to find it.

## Architecture

- `internal/provider` — the `Provider` contract: `Request` (prompt + material by value), `Result` (findings + provenance), sentinel errors, `HashPrompt`.
- `internal/provider/providertest` — the shared conformance suite (`Conform`) and the `Loopback` reference provider that validates it.

## Milestones

- [ ] `bootstrap-release` — `scripts/release`, `.github/workflows/`, `CHANGELOG.md`, tap formula.
- [x] Provider interface and conformance suite.
- [ ] Codex adapter.
- [ ] CLI front-end; retire `dotfiles/claude/bin/second-opinion.sh`.
- [ ] MCP front-end.
- [ ] `gemini` and `ollama` adapters.

## Release

Added by `bootstrap-release`, Go / github-tarball flavor.
`scripts/release vX.Y.Z` promotes `## [Unreleased]`, tags, and pushes; GitHub Actions updates `homebrew-tap`.
