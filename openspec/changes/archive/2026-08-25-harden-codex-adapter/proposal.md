## Why

The Codex adapter currently prevents some implicit context leaks, but its isolation claim is stronger than the command line can guarantee.
It also treats incomplete or changed JSONL output as a successful empty review, which can hide provider protocol drift.

This change makes the current Codex integration honest and fail-closed before it becomes a foundation for MCP or stricter material-only execution.

## What Changes

- Suppress Codex user configuration and execution-policy rules during reviews.
- Preserve the current “no implicit context” behavior while documenting that strict filesystem isolation is a separate capability.
- Fail a review when Codex emits an incomplete or unrecognized successful event stream.
- Harden nested error extraction, authentication classification, model fallback, and cancellation behavior.
- Add material identity to review provenance so callers can verify what was reviewed.
- Expand deterministic fixtures and real-binary conformance coverage for the current Codex event protocol.
- Update the Codex adapter documentation, milestone state, and README claims to match verified behavior.

## Non-goals

- Enforcing true material-only filesystem access for Codex subprocesses.
- Implementing the MCP front-end.
- Enforcing same-model author/reviewer policy.
- Changing the raw findings format or adding ranking and triage logic.
- Adding new providers.

## Capabilities

### New Capabilities

- `codex-provider-hardening`: Make the Codex adapter’s isolation posture, event parsing, provenance, and failure behavior explicit and fail-closed.

### Modified Capabilities

- `provider-interface`: Add material identity to successful review provenance so the assembled request remains auditable.

## Impact

- `internal/provider/provider.go` and provider conformance tests.
- `internal/provider/codex` implementation, parser, unit fixtures, and integration tests.
- Codex adapter OpenSpec artifacts and project documentation, including `README.md` and `AGENTS.md` milestone notes.
- No new runtime dependency is expected; behavior depends on supported flags and JSONL output from the installed `codex` binary.
