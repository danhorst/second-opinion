# provider-interface

## Why

Every planned feature — the CLI, the MCP gateway, and all three adapters — sits on top of one seam: the interface between the review engine and the model that performs the review.
Defining that seam first, together with the conformance suite that proves the cold-reviewer guarantee, means every adapter that follows is built against a contract instead of against the first adapter's accidents.

## What Changes

- Bootstrap the Go module (`go.mod`, `internal/` layout, `make test`).
- Define the `Provider` interface in `internal/provider`: a review request (prompt + explicitly assembled material) in, a review result (findings + reviewer provenance) out.
- Define the request and result types the interface speaks, including the provenance fields (provider, model, prompt identity) that must survive to the caller.
- Define error semantics that distinguish "the reviewer ran" from "the reviewer did not run".
- Ship the shared conformance suite (`internal/provider/providertest`) that every adapter must pass, including the cold-reviewer proof: the reviewer receives nothing the caller did not explicitly assemble.
- Validate the suite itself against an in-process reference provider, so the suite has a consumer before the first real adapter lands.

## Capabilities

### New Capabilities
- `provider-interface`: the contract between the review engine and any reviewer — request shape, result shape with provenance, error semantics, transport-agnosticism.
- `provider-conformance`: the shared test suite that proves an adapter honors the contract, above all the cold-reviewer guarantee.

### Modified Capabilities
None — `openspec/specs/` is empty; this is the first change.

## Non-goals

- The `codex`, `claude`, `gemini`, and `ollama` adapters (each is its own change; the codex adapter is the suite's first real consumer).
- The CLI and MCP front-ends.
- Target assembly (file concatenation, `git diff`) — the interface takes material already assembled; how it gets assembled is a front-end concern.
- Triage, clustering, or ranking of findings — open design question, deliberately untouched here.
- Model selection and auth fallback — provider-specific behavior, specified with each adapter.
- Same-model author/reviewer enforcement — this change makes it *detectable* by pinning provenance into the result; enforcement is a later decision.
- Release wiring (`bootstrap-release` runs separately).

## Impact

- New Go module rooted at the repo; new packages `internal/provider` and `internal/provider/providertest`.
- No existing code is affected — the repo has none.
- Every subsequent change depends on the types defined here; getting the result's provenance fields right now is what keeps the reviewer-≠-author premise auditable later.
