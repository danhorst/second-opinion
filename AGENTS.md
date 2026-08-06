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
- A passing cold result is meaningful only alongside a passing positive control.
  Every cold check is a negative assertion, so a reviewer that received no material satisfies all of them.
  The suite runs the control first and reports the cold checks as inconclusive beneath a failed one.
- Do not mock the provider boundary to reach a coverage number.
- Anything genuinely untestable is named in this file. An untested path nobody wrote down is a bug waiting for a user to find it.

## Architecture

- `internal/provider` — the `Provider` contract: `Request` (prompt + material by value), `Result` (findings + provenance), sentinel errors, `HashPrompt`.
- `internal/provider/providertest` — the shared conformance suite (`Conform`) and the `Loopback` reference provider that validates it.
- `internal/provider/codex` — the codex adapter: cold invocation by construction, JSONL provenance extraction, ChatGPT-auth model fallback.
- `internal/provider/antigravity` — the antigravity (`agy`) adapter: cwd isolation, prompt-embedded material, documented provenance degradation for unforced reviews.
- `internal/provider/claude` — the claude adapter: tools and setting sources disabled (repo reads impossible, user memory suppressed), truthful model provenance from the JSON envelope.
- `internal/review` — engine pieces shared by front-ends: the baked adversarial prompt (transport-neutral), target assembly (`FromFiles`, `FromDiff`), the provider registry.
- `cmd/second-opinion` — the CLI: findings on stdout, provenance on stderr; exit 0 = review completed, 1 = reviewer did not run, 2 = usage. `skill install` writes the calling-agent skill per harness.
- `skills/second-opinion/SKILL.md` — the canonical calling-agent skill (triage discipline), embedded in the binary.

## Milestones

- [x] `bootstrap-release` — `scripts/release`, `.github/workflows/`, `CHANGELOG.md`, tap formula; v0.1.0 released 2026-07-14.
- [x] Provider interface and conformance suite — the positive control landed 2026-08-06, so the cold checks no longer pass vacuously against a provider that sends no material.
- [ ] Codex adapter — passes full real-binary conformance including the positive control, 2026-08-06; model provenance is still broken (see Known defects).
- [x] CLI front-end — implemented; `dotfiles/claude/bin/second-opinion.sh` has been retired.
- [ ] MCP front-end.
- [x] Antigravity (`agy`) adapter — the 2026-07-10 pass was void, against a suite that could not detect the adapter sending no material; the invocation is fixed and it passes under the positive control, 2026-08-06.
- [x] Claude adapter — the 2026-07-10 pass was against that same suite; re-verified under the positive control, 2026-08-06.
- [x] Calling-agent skill, shipped in the binary (`skill install`).
- [ ] `ollama` adapter (the planned `gemini` adapter was dropped: Google deprecated the Gemini CLI in favor of Antigravity, which the roster already carries).

## Known defects

Found 2026-08-06, by dogfooding a review of `openspec/changes/findings-envelope/`.

**Codex provenance reports `codex-default` on a successful review — unowned.**
The review ran and returned real findings, but `findModel` (`internal/provider/codex/events.go`) found no string-valued `model` key anywhere in the JSONL event stream, so provenance fell back to the marker.
Reproduce with `second-opinion --provider codex <file>` and read stderr.
Most likely codex changed its `--json` output shape, or `--ephemeral` suppresses the `session_configured` event; a real `findModel` bug is the third possibility.
This matters because provenance is the only mechanism that makes a reviewer-equals-author violation detectable after the fact — a provider that cannot name the model that ran defeats it.
Named as an explicit non-goal in `openspec/changes/conformance-positive-control/`, so no change owns it.
`make test-integration` fails on `TestCodexProvenance` for this reason alone; conformance passes for all three adapters.

## Release

Added by `bootstrap-release`, Go / github-tarball flavor.
`scripts/release vX.Y.Z` promotes `## [Unreleased]`, tags, and pushes; GitHub Actions updates `homebrew-tap`.
