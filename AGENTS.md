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

Current verification evidence and limits are recorded in `docs/reviews/2026-11-13-hardening.md`:
Codex and Antigravity passed the positive control and relative/absolute-path cold checks on
2026-11-13. Claude's stronger invocation could not be certified because OAuth expired;
its cold checks were inconclusive beneath a failed positive control. Full harness-global
isolation remains an evidence gap, especially for Antigravity. Real subprocess tests
cover active cancellation and inherited-pipe cleanup on the host Unix platform; detached
children outside the process group and Windows descendant cleanup remain uncovered.
The shared Provider suite still checks pre-cancellation: it has no readiness signal to
prove an arbitrary reviewer is active. These are follow-ups, not conformance exemptions.
Review quality, completeness, and model obedience cannot be proved by deterministic tests.
MCP transport and real-provider evidence is recorded in `docs/reviews/2026-11-14-mcp.md`;
MCP does not strengthen the provider isolation guarantees above.

## Architecture

- `internal/provider` — the `Provider` contract: `Request` (prompt + material by value), `Result` (findings + provenance), sentinel errors, `HashPrompt`.
- `internal/provider/process` — shared subprocess group cleanup, bounded output drain, and actionable OS argument-limit errors.
- `internal/provider/providertest` — the shared conformance suite (`Conform`) and the `Loopback` reference provider that validates it.
- `internal/provider/codex` — the codex adapter: explicit context/tool suppression, validated terminal JSONL results, documented provenance degradation for unforced reviews, ChatGPT-auth model fallback.
- `internal/provider/antigravity` — the antigravity (`agy`) adapter: cwd isolation, prompt-embedded material, documented provenance degradation for unforced reviews.
- `internal/provider/claude` — the claude adapter: safe mode, tools/settings/skills/MCP suppressed, stdin request transport, validated JSON results and model usage provenance; latest real-binary validation blocked on OAuth.
- `internal/mcpserver` — stateless review/discovery tools, strict schemas, bounded stdio and authenticated loopback HTTP transports; no repository reads, history, or result retrieval.
- `internal/review` — one-shot executor with fresh provider construction and result validation; engine pieces shared by front-ends: the baked adversarial prompt (transport-neutral), target assembly (`FromFiles`, `FromDiff`), the provider registry.
- `cmd/second-opinion` — the CLI: text findings or `--json` results, provenance on stderr; `--dry-run` previews requests, `--timeout` bounds provider calls. Exit 0 = operation succeeded, 1 = review/output failure, 2 = usage. `skill install` writes the calling-agent skill per harness.
- `skills/second-opinion/SKILL.md` — the canonical calling-agent skill (triage discipline), embedded in the binary.

## Milestones

- [x] `bootstrap-release` — `scripts/release`, `.github/workflows/`, `CHANGELOG.md`, tap formula; v0.1.0 released 2026-07-14.
- [x] Provider interface and conformance suite — the positive control landed 2026-08-06, so the cold checks no longer pass vacuously against a provider that sends no material.
- [x] Codex adapter — passes full real-binary conformance including the positive control, 2026-08-06; provenance degradation for unforced reviews verified against the real binary, 2026-08-07.
- [x] CLI front-end — implemented; `dotfiles/claude/bin/second-opinion.sh` has been retired.
- [x] MCP front-end — stateless stdio and private loopback HTTP; verification and inherited limits in `docs/reviews/2026-11-14-mcp.md`.
- [x] Antigravity (`agy`) adapter — the 2026-07-10 pass was void, against a suite that could not detect the adapter sending no material; the invocation is fixed and it passes under the positive control, 2026-08-06.
- [x] Claude adapter — the 2026-07-10 pass was against that same suite; re-verified under the positive control, 2026-08-06.
- [x] Calling-agent skill, shipped in the binary (`skill install`).
- [ ] `ollama` adapter (the planned `gemini` adapter was dropped: Google deprecated the Gemini CLI in favor of Antigravity, which the roster already carries).

## Release

Added by `bootstrap-release`, Go / github-tarball flavor.
`scripts/release vX.Y.Z` promotes `## [Unreleased]`, tags, and pushes; GitHub Actions updates `homebrew-tap`.
