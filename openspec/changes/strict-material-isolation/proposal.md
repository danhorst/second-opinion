## Why

Cold invocation flags and passing canaries do not establish a filesystem read boundary. Callers need an enforceable, narrowly named requirement without implying that this client controls a remote endpoint's internal context.

## What Changes

- Define `local-material-only`: after caller assembly, the adapter sends only prompt/material as review content, does not load ambient content, execute reviewer tools, dereference reviewer-supplied paths/URLs, or launch a reviewer harness. Credentials and transport configuration are trusted operational inputs, not review content.
- Expose optional `--require-isolation local-material-only` and MCP `require_isolation`; unsupported requirements fail before reviewer execution, without fallback.
- Report the effective adapter boundary in review results and provider discovery. Unknown and subprocess adapters report `unverified`, which does not revoke their existing cold-invocation obligations.
- Support the boundary in the OpenAI-compatible adapter. Reject tool requests and redirects. Trust the configured endpoint and HTTP transport explicitly; do not call this endpoint or model certification.
- Test local transport and policy enforcement deterministically, retaining real-provider conformance as separate evidence.

## Non-goals

OS sandboxing of CLI harnesses; endpoint-side isolation or model obedience certification; protection from compromised local processes, endpoints, transports, or credentials; local-only inference; ranking or author-model enforcement; changing the content-only provider request.

## Capabilities

### New Capabilities
- `strict-material-isolation`: A scoped local material boundary, enforceable requirements, discovery, receipts, and evidence limits.

### Modified Capabilities
- `provider-interface`: Optional transport-neutral isolation capability and fail-closed execution policy separate from review content.
- `review-cli`: Isolation requirement, preview, result, and discovery fields.
- `mcp-review`: Isolation requirement, discovery, results, and not-started errors.
- `openai-compatible-provider`: Material-only client boundary and explicit rejection of redirects and tool requests.

## Impact

Provider policy helpers, HTTP adapter, review executor, CLI, MCP schemas and validation, tests, README and evidence documentation. No new dependencies; existing default provider selection and plain provider listing remain compatible.
