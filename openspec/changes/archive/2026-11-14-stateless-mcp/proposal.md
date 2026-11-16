## Why

Coding harnesses should be able to request a second opinion without shell parsing or coupling review identity to a long-lived server session. The existing provider contract already accepts complete input by value and returns raw findings with provenance; MCP can expose it without waiting for heuristic findings parsing.

## What Changes

- Add `second-opinion mcp serve` with stdio by default and opt-in loopback-only stateless Streamable HTTP.
- Expose `review` and `list_providers` tools. Each review supplies its provider and full material, with optional model, prompt override, and timeout.
- Return raw findings and full provenance together in versioned structured content and equivalent JSON text, with machine-readable tool errors.
- Retain no conversations, results, jobs, deduplication records, or review history. Only bounded in-flight execution and transport bookkeeping exist.
- Define request, output, time, concurrency, cancellation, and local HTTP security boundaries.
- Keep findings parsing, ranking, and novelty out of the MCP contract; make the existing findings-envelope proposal optional rather than a prerequisite.

## Capabilities

### New Capabilities

- `mcp-review`: stateless review tools, transports, output/error contracts, and operational limits.

### Modified Capabilities

- `provider-interface`: opt-in bounded output capture without changing content-only requests.
- `agent-skill`: teach MCP invocation, explicit material selection, provenance, and retry discipline.

## Non-goals

remote/public hosting; multi-user credential delegation; Ollama; findings parsing; same-model enforcement; triage/ranking; persistence; async task APIs; result retrieval; exactly-once billing; automatic tool retries; repository/file/URL tools; client sampling, roots, elicitation, or resource subscriptions.

## Impact

Implementation will add the official Go MCP SDK (stable 1.x release pinned at implementation), a transport adapter under `internal/mcpserver`, a shared engine executor, CLI dispatch, harness documentation, and contract tests. Existing CLI review behavior remains compatible. This proposal does not sync into current implementation specs until applied.
