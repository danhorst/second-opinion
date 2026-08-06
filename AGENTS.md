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
- [ ] Provider interface and conformance suite — implemented, but the suite's cold checks pass vacuously against a provider that sends no material; `conformance-positive-control` closes that.
- [ ] Codex adapter — implemented; real reviews run (usage limit reset 2026-07-19 has passed), but the suite has not been re-run since, and model provenance is broken (see Known defects).
- [x] CLI front-end — implemented; `dotfiles/claude/bin/second-opinion.sh` has been retired.
- [ ] MCP front-end.
- [ ] Antigravity (`agy`) adapter — the 2026-07-10 conformance pass is void: the adapter has never sent material to the reviewer (see Known defects). Blocked on `conformance-positive-control`.
- [ ] Claude adapter — the 2026-07-10 conformance pass was against a suite that could not detect a provider sending no material. A real review on 2026-08-06 demonstrably received its material, so the defect is not shared; pending re-verification under the positive control.
- [x] Calling-agent skill, shipped in the binary (`skill install`).
- [ ] `ollama` adapter (the planned `gemini` adapter was dropped: Google deprecated the Gemini CLI in favor of Antigravity, which the roster already carries).

## Known defects

Both found 2026-08-06, by dogfooding a review of `openspec/changes/findings-envelope/`.

**Codex provenance reports `codex-default` on a successful review — unowned.**
The review ran and returned real findings, but `findModel` (`internal/provider/codex/events.go`) found no string-valued `model` key anywhere in the JSONL event stream, so provenance fell back to the marker.
Reproduce with `second-opinion --provider codex <file>` and read stderr.
Most likely codex changed its `--json` output shape, or `--ephemeral` suppresses the `session_configured` event; a real `findModel` bug is the third possibility.
This matters because provenance is the only mechanism that makes a reviewer-equals-author violation detectable after the fact — a provider that cannot name the model that ran defeats it.
Named as an explicit non-goal in `openspec/changes/conformance-positive-control/`, so no change owns it.

**Antigravity sends no material — owned by `conformance-positive-control`.**
`agy --print` takes the prompt as its flag value, and `buildInvocation` emits `agy --print --sandbox … <instruction>`, so `--print` consumes `--sandbox` and the material is dropped.
The reviewer answers the string `--sandbox` and the tool exits 0.
Every cold-guarantee check passes vacuously, because a reviewer that receives nothing cannot see anything it should not.

## Release

Added by `bootstrap-release`, Go / github-tarball flavor.
`scripts/release vX.Y.Z` promotes `## [Unreleased]`, tags, and pushes; GitHub Actions updates `homebrew-tap`.
