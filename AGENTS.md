Adversarial review: send a document or a diff to a model that did not write it, and get findings back.
The tool has no home model — it is callable from any harness to any provider, and Claude is a provider like any other.

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

- `internal/provider` — the `Provider` contract: `Request` (prompt + material by value), `Result` (findings + provenance), sentinel errors, `HashPrompt`, and `HashMaterial`.
- `internal/provider/providertest` — the shared conformance suite (`Conform`) and the `Loopback` reference provider that validates it.
- `internal/provider/codex` — the codex adapter: implicit-context suppression, JSONL provenance extraction, ChatGPT-auth model fallback, and explicit `codex-default` degradation when unforced model identity is absent.
- `internal/provider/openai_compatible` — the configurable non-streaming HTTP adapter for OpenAI-compatible endpoints, including OpenRouter.
- `internal/provider/antigravity` — the antigravity (`agy`) adapter: cwd isolation, prompt-embedded material, documented provenance degradation for unforced reviews.
- `internal/provider/claude` — the claude adapter: tools and setting sources disabled (repo reads impossible, user memory suppressed), truthful model provenance from the JSON envelope.
- `internal/review` — engine pieces shared by front-ends: the baked adversarial prompt (transport-neutral), target assembly (`FromFiles`, `FromDiff`), the provider registry.
- `cmd/second-opinion` — the CLI: findings on stdout, provenance on stderr; exit 0 = review completed, 1 = reviewer did not run, 2 = usage. `skill install` writes the calling-agent skill per harness.
- `skills/second-opinion/SKILL.md` — the canonical calling-agent skill (triage discipline), embedded in the binary.

## Milestones

- [x] `bootstrap-release` — `scripts/release`, `.github/workflows/`, `CHANGELOG.md`, tap formula; v0.1.0 released 2026-07-14.
- [x] Provider interface and conformance suite.
- [x] Codex adapter — implemented; real-binary conformance verified against codex-cli 0.149.0-alpha.4.3 on 2026-08-25.
- [ ] OpenAI-compatible provider — implemented; endpoint integration verification depends on configured API credentials.
- [ ] CLI front-end — implemented; retiring `dotfiles/claude/bin/second-opinion.sh` is a manual follow-up in the dotfiles repo once the binary is installed.
- [ ] MCP front-end.
- [x] Antigravity (`agy`) adapter — full real-binary conformance passed 2026-07-10.
- [x] Claude adapter — full real-binary conformance passed 2026-07-10.
- [x] Calling-agent skill, shipped in the binary (`skill install`).
- [ ] `ollama` adapter (the planned `gemini` adapter was dropped: Google deprecated the Gemini CLI in favor of Antigravity, which the roster already carries).

## Release

Added by `bootstrap-release`, Go / github-tarball flavor.
`scripts/release vX.Y.Z` promotes `## [Unreleased]`, tags, and pushes; GitHub Actions updates `homebrew-tap`.
