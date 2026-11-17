## Context

See proposal.md. The current CLI calls providers directly while MCP uses the shared executor. Request content must remain exactly prompt and material. Existing working-tree changes are unrelated and must be preserved.

## Goals / Non-Goals

**Goals:** One policy gate shared by CLI and executor; capability determined from the constructed provider; explicit and auditable local scope.

**Non-Goals:** Portable process sandboxing, endpoint certification, or weakening existing cold invocation controls.

## Decisions

1. Name the supported boundary `local-material-only`, not `strict`. `unverified` means no certified local boundary, not permission to fetch context. The threat is instructions in untrusted material or a response asking the client to read local data. The client code, HTTP transport, credentials, and configured endpoint are trusted. An endpoint with its own filesystem access is outside this boundary.
2. An optional provider capability interface reports the boundary; absence or unknown values become `unverified`. A shared execution helper checks the requirement before calling Review and stamps successful results with the effective boundary. It does not alter provenance. Requirements remain execution policy, outside provider.Request. Unsupported/unknown requirements produce a sentinel error, no result, no execution, and no fallback.
3. CLI exposes `--require-isolation local-material-only`. Preview records a requested requirement without claiming it was satisfied. JSON review results and stderr report the effective boundary. Existing plain provider listing stays stable; JSON listing includes names and isolation metadata. MCP adds matching optional input, result metadata and provider discovery mapping. Discovery uses static registration metadata without reading credentials; execution independently checks the constructed provider, and a test verifies metadata consistency. Unsupported requirements report `isolation_unsupported` / `not_started`.
4. The HTTP adapter sends exactly system prompt and user material; no tool definitions, local reads, or tool execution loop. Copy the configured HTTP client and disable redirects, including for injected clients. Reject tool/function calls even if accompanied by text. Preserve unknown endpoint fields for compatibility by ignoring them, never executing them. Configuration/credentials and custom transports remain trusted. This makes a client-side guarantee, not a claim of remote endpoint certification.
5. Test policy using reference providers and HTTP mechanics using local fixtures. Fixtures are explicitly not model-conformance evidence. Exercise hostile paths, URL/tool requests and redirect denial. Keep existing real-provider positive-control-gated conformance separate. A canary's absence cannot prove inability to access a filesystem.

## Cold-reviewer guarantee

Existing harness suppression and conformance remain required. The new path strengthens the locally enforceable part through absence of a harness/tool loop. It cannot certify endpoint-added context or global OS isolation; document those limits rather than infer them from cold checks.

## Risks / Trade-offs

- Endpoint trust may be mistaken for whole-system isolation → scoped name and explicit documentation in help/schema/evidence.
- Optional capabilities can be falsely advertised by future code → unknown defaults closed; transport tests and code review are required for new support.
- Redirect rejection may break redirecting endpoints → require the final endpoint URL; no silent weakening.
- Live endpoint credentials may be unavailable → record missing certification explicitly; local tests certify only client mechanics.

## Migration Plan

Additive opt-in policy and output metadata; no default provider changes. Redirects and tool-call responses now fail explicitly. Rollback removes the opt-in fields and capability without changing assembled review content.
