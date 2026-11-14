## Why

The provider adapters currently can report success when Codex's event stream is malformed, truncated, oversized, or contains no completed review, and when Claude returns an empty or non-result JSON value. These false successes erase the distinction between an empty review and a review that never produced a valid result.

## What Changes

- Require Codex JSONL output to be well formed and end with a recognized successful completion event before returning findings.
- Accept an explicitly completed empty findings message, while rejecting missing findings and failed/interrupted turns.
- Require Claude output to be a successful result envelope with a present string result; preserve result bytes, including leading and trailing whitespace.
- Keep valid nonfatal Codex item warnings compatible with result extraction.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `codex-provider`: specify validation of completed event streams and findings.
- `claude-provider`: specify validation of successful result envelopes and raw result preservation.

## Impact

Changes are limited to `internal/provider/codex` and `internal/provider/claude`, their provider specs, and deterministic parser/envelope tests. Provider process invocation and lifecycle are unchanged; no new dependencies or provider-boundary mocks are introduced.

## Non-goals

- Changing subprocess cancellation, timeout, or descendant-process cleanup.
- Changing the provider interface or adding adapters.
- Establishing review quality or whether a model follows an output convention.
