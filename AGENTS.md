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
`mcp init` filesystem tests cover private creation, existing-token validation, atomic
concurrent initialization, and secret-free output.
Homebrew launcher tests cover token-file validation, environment/argv delivery, and
exit propagation; the generated service metadata is checked with Homebrew. Live
launchd/systemd lifecycle and provider credentials under a service environment have
not been verified on this machine.

MCP harness setup evidence is recorded in `docs/reviews/2026-11-16-mcp-harness-setup.md`.
MCP harness setup tests cover temporary configuration edits, backups, idempotence,
conflicts, private token handling, and real local MCP discovery. Live Codex/Claude Code/
Antigravity acceptance, GUI reload/trust flows, and managed-policy precedence have not
been verified on this machine. Doctor verifies protocol discovery, not reviewer credentials.
Configuration updates detect changed content before publication but do not provide a
cross-process lock against concurrent harness writes.

## Architecture

- `internal/provider` — the `Provider` contract: `Request` (prompt + material by value), `Result` (findings + provenance), sentinel errors, `HashPrompt`, and `HashMaterial`.
- `internal/provider/process` — shared subprocess group cleanup, bounded output drain, and actionable OS argument-limit errors.
- `internal/provider/providertest` — the shared conformance suite (`Conform`) and the `Loopback` reference provider that validates it.
- `internal/provider/codex` — the codex adapter: explicit context/tool suppression, validated terminal JSONL results, documented provenance degradation for unforced reviews, ChatGPT-auth model fallback.
- `internal/provider/openai_compatible` — configurable HTTP adapter for OpenAI-compatible endpoints; endpoint integration requires configured credentials.
- `internal/provider/antigravity` — the antigravity (`agy`) adapter: cwd isolation, prompt-embedded material, documented provenance degradation for unforced reviews.
- `internal/provider/claude` — the claude adapter: safe mode, tools/settings/skills/MCP suppressed, stdin request transport, validated JSON results and model usage provenance; latest real-binary validation blocked on OAuth.
- `internal/harness` — safe opt-in MCP configuration adapters for Codex, Claude Code, and Antigravity.
- `internal/mcpclient` — bounded private stdio-to-HTTP connector; no review history.
- `internal/mcpcheck` — non-review initialization and tool discovery for diagnostics.
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
- [ ] OpenAI-compatible endpoint certification — implemented; configured endpoint verification remains required.
- [ ] `ollama` adapter (the planned `gemini` adapter was dropped: Google deprecated the Gemini CLI in favor of Antigravity, which the roster already carries).

## Release

`packaging/homebrew/second-opinion.rb.in` is the canonical formula template;
`scripts/render-homebrew-formula` stamps the release tag/checksum. The formula installs
`scripts/second-opinion-mcp-service` as an optional per-user HTTP service launcher.

Added by `bootstrap-release`, Go / github-tarball flavor.
Release only from clean `main` exactly matching freshly fetched `origin/main`. Merge and
validate feature work before running `scripts/release vX.Y.Z`; the script checks this
before modifying anything and atomically pushes the release commit and tag. The
publishing workflow independently rejects tags outside main history or with a mismatched
embedded version. Never release directly from a feature branch or move a published tag.
See `docs/releasing.md` for the complete procedure and failure recovery.
