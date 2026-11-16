# Stateless MCP review gateway

## Context

The provider contract is already one-shot: `Review(ctx, Request{Prompt, Material})` returns raw text and provenance. CLI target assembly is a caller convenience; it must not become implicit filesystem access inside an MCP server. The CLI JSON shape provides a useful success envelope without requiring the pending findings parser.

The preceding hardening implementation was committed as `8cdd37a`. This change is now being applied; its acceptance evidence is recorded in `docs/reviews/2026-11-14-mcp.md`.

## Goals / Non-Goals

Provide a predictable, stateless service callable from coding harnesses with bounded resource usage and unchanged cold-review semantics. Keep the tool surface small and make a lost response distinguishable from a known failure to start.

No stored reviews, sessions containing review data, job handles, caches, deduplication, resume, conversation history, automatic retries, roots, file/URL resolution, client sampling, elicitation, resource subscriptions, or triage. No public network deployment or per-user provider credentials. MCP does not depend on the findings-envelope parser or change the baked prompt.

## Decisions

### D1. Stateless application, transient execution

Each call contains all review-specific inputs. Provider/model/prompt never inherit a previous call or the MCP client's identity. `SECOND_OPINION_PROVIDER` is not a default for MCP. Server startup configuration and existing operator-owned provider authentication are permitted deployment configuration, not review state.

Only active request contexts, cancellation routing, transport metadata, and a fixed-capacity execution semaphore survive beyond one stack frame. They are released on completion/disconnection. There is no application store keyed by client, connection, request ID, material hash, or review ID. A restart loses in-flight calls; nothing can be resumed or retrieved.

The MCP process does not persist or log prompts, material, raw output, tokens, or hashes. This is not a claim that upstream model services or third-party harnesses never retain data. Existing adapter temporary directories retain their cleanup behavior. Go garbage collection is not secure memory erasure.

### D2. Two tools, complete inputs

`review` input is a closed object (`additionalProperties: false`):

| Field | Required | Meaning |
| --- | --- | --- |
| `provider` | yes | Exact registered provider name; no default |
| `material` | yes | Complete UTF-8 text: document(s), caller-generated patch, or deliberately bundled context |
| `model` | no | Nonempty forced model name; provider fallback semantics unchanged |
| `prompt` | no | Non-whitespace text replacing the baked prompt; omission selects `review.BakedPrompt` |
| `timeout_ms` | no | Integer 1–300000; omission selects 120000 |

Reject missing, null, wrongly typed, blank required text, unknown fields, or excessive values before constructing/invoking a provider. Do not trim or normalize accepted strings. Provider names are validated against the registry. Model names are at most 256 UTF-8 bytes. Whitespace-only material is invalid. A string containing a path is text to review, never a path to open.

`list_providers` accepts `{}` and returns sorted registered names plus the effective limits described in D6. It does not probe binaries, authentication, model catalogs, or account quota. Discovery never spends a model call. Provider availability errors surface only when reviewing. Its success object is `{"schema_version":1,"kind":"providers","providers":["antigravity","claude","codex"],"limits":{...}}`; the list is registry-derived, not hardcoded. Limit keys are `max_request_bytes`, `max_input_bytes`, `max_model_bytes`, `default_timeout_ms`, `max_timeout_ms`, `max_concurrent_reviews`, `max_stdout_bytes`, `max_stderr_bytes`, `max_raw_bytes`, `max_response_bytes`, and `max_inflight_messages`. It uses the same structured-content/JSON-text mirroring rule as review.

No `review_files`, `review_diff`, or `get_result` tools. A coding harness already controls its workspace and can assemble exactly the intended input. Multiple files can be passed in one material string with caller-selected labels. Missing dependencies stay missing; the reviewer cannot fetch them.

### D3. Complete outcomes, raw text authoritative

Success structured content:

```json
{
  "schema_version": 1,
  "kind": "review",
  "raw": "reviewer text, including an empty string",
  "provenance": {
    "provider": "codex",
    "model": "codex-default",
    "prompt_hash": "<full SHA-256 of the resolved prompt>"
  },
  "request": {
    "material_hash": "<SHA-256 of exact UTF-8 material bytes>",
    "material_bytes": 123,
    "timeout_ms": 120000
  }
}
```

The input receipt does not store content or establish finding identity; it lets the caller tie a result to its own retained input. Hashes never key a cache. Provider provenance is copied unaltered; a malformed/nil provider result becomes an error. `raw` is precisely provider-returned text, not a claim to undo transport transformations already performed by an adapter. Empty raw is successful only after adapter transport validation.

The MCP result contains this object in `structuredContent` and exactly one text block containing its JSON serialization. This gives text-only callers the same provenance and avoids appending a second, differently interpreted findings summary. Both represent the same value. Set `isError=false`. The SDK emits protocol-version-specific framing, including modern `resultType`; application code does not hardcode protocol framing.

`review` declares an output schema covering both success and failure object variants. Failure:

```json
{
  "schema_version": 1,
  "kind": "error",
  "error": {
    "code": "timeout",
    "message": "Review exceeded its deadline; completion is unknown.",
    "execution": "unknown"
  }
}
```

Use `isError=true` and the same structured-plus-JSON-text representation. Do not include empty success fields or fabricated provenance. Error messages are fixed, actionable summaries; never forward arbitrary provider stderr, which may contain credentials or material. Safe details include the selected registered provider and numeric limits, not prompt text, paths from errors, or output fragments.

| Code | Trigger | `execution` |
| --- | --- | --- |
| `invalid_input` | Unknown provider, invalid arguments, whitespace-only input | `not_started` |
| `input_too_large` | Decoded text budget exceeded | `not_started` |
| `busy` | No execution slot | `not_started` |
| `unavailable` | `provider.ErrUnavailable` | `unknown` (quota and missing executable share this sentinel) |
| `auth` | `provider.ErrAuth` | `unknown` |
| `timeout` | Server review deadline | `unknown` |
| `cancelled` | Cancellation when a response is still permitted | `unknown` |
| `output_too_large` | Provider capture or result limit exceeded | `unknown` |
| `provider_failed` | Malformed transport, other provider error, invalid result | `unknown` |

Malformed JSON-RPC or unknown methods/tools remain SDK protocol errors. Tool argument/domain validation is a tool error; if SDK schema validation would bypass this envelope, validate raw tool arguments in a lower-level handler. HTTP body/auth rejection occurs before a tool call and uses HTTP status, not a fake tool result.

No `retryable` boolean: the existing errors cannot guarantee that a provider did no work or incurred no charge. No gateway retry. The Codex adapter's existing single auth-model fallback remains within the same deadline. A client retry is a new independent review that may repeat work and spend money. Request IDs correlate active messages; they are not idempotency keys.

### D4. Transport and local deployment

```text
second-opinion mcp serve
second-opinion mcp serve --transport http --listen 127.0.0.1:8765
```

Stdio is the default. Only MCP messages go to stdout; bounded operational diagnostics go to stderr. A successful tool failure does not exit the process. Invalid startup flags exit 2; startup/transport failures exit 1; normal client shutdown exits 0. EOF or SIGTERM cancels outstanding work and waits at most five seconds for cleanup before process exit.

HTTP uses one `/mcp` endpoint and the official Go SDK's `StreamableHTTPOptions{Stateless: true}` with no event store, session ID persistence, replay, or resumability. Prefer completed JSON responses rather than application-level streaming. Do not expose a persistent server-event GET stream. Every HTTP request supplies its authorization and protocol metadata; no sticky routing is needed. Per-process concurrency limits are not a distributed quota.

Support MCP `2026-07-28` and the SDK's `2025-11-25` compatibility path, tested explicitly. Modern requests carry protocol metadata per request; legacy stdio initialization retains only protocol negotiation metadata. Legacy HTTP must also work without application/session affinity under the stateless handler. If a chosen SDK version cannot satisfy this, do not silently enable stateful HTTP: adjust the supported-version list explicitly and record the compatibility change. The implementation pins official Go SDK v1.8.0 and tests these versions; it does not track `main`.

HTTP is single-operator loopback only: bind literal `127.0.0.1` or `::1`; reject wildcard, hostname, and non-loopback listeners. Require `SECOND_OPINION_MCP_TOKEN` with at least 32 bytes, supplied outside argv. The operator generates it randomly and configures the harness to send `Authorization: Bearer ...`; no generated-token store or login flow is added. Compare tokens in constant time, fail missing/invalid auth before parsing tool input, never log or pass the token to provider processes. Remove this gateway-only variable from child environments. Provider credentials remain server-owned and are never accepted as tool inputs.

Reject browser `Origin` headers (including `null`) and validate Host against the literal bound authority. No permissive CORS, redirects, proxy trust, or public binding escape hatch. This is a private local capability token, not a claimed implementation of public MCP OAuth discovery. Stdio relies on OS process access and does not require this token. Public serving would require a separate authorization/deployment design.

### D5. Cancellation without a job registry

The execution context is the earliest of client cancellation/disconnect, parent shutdown, and `timeout_ms`. It reaches the provider and its existing process-group cleanup. Release the slot on every path, including panic recovery at the server boundary. Do not let cancellation of one call cancel a sibling.

For stdio, the SDK routes cancellation to the active request; stop output for a cancelled modern request as required by that transport. EOF cancels all calls on that connection. For HTTP, cancelling/closing the originating response stream cancels that request. Do not add a cross-request cancellation lookup or accept a session ID as a result handle. Legacy HTTP callers must close the original request to cancel rather than depend on a stateful cancellation notification.

A server-originated deadline returns a timeout tool error while the connection is usable. An explicit client cancellation or broken connection may yield no result at all; the design does not promise delivery after disconnect. The caller must treat this as unknown completion and cannot query the server later.

### D6. Concrete limits, no waiting queue

Defaults are fixed in v1 and returned by `list_providers`; no per-call expansion:

| Limit | Value | Enforcement |
| --- | --- | --- |
| Transport request frame/body | 8 MiB | Before JSON decoding; oversize HTTP is 413; oversize stdio closes with bounded diagnostic |
| Material plus resolved prompt | 1 MiB UTF-8 bytes | Before provider construction; no truncation |
| Model name | 256 UTF-8 bytes | Before provider construction |
| Review timeout | default 120 s, maximum 300 s | Full provider call including fallback |
| Concurrent reviews | 2 per server process | Nonblocking acquire; additional calls return `busy` |
| Provider stdout capture | 8 MiB per invocation | Capped writer cancels child on overflow; no partial findings |
| Provider stderr capture | 64 KiB per invocation | Capped writer cancels child on overflow; no arbitrary diagnostics returned |
| Raw findings | 1 MiB UTF-8 bytes | Before MCP serialization; no truncation |
| Encoded MCP response | 16 MiB | Before write, allowing JSON escaping and duplicated structured/text values |

Ingress bounds apply to discovery as well. Validate UTF-8 and reject lossy decoding (including malformed Unicode escapes) rather than silently replacing bytes. Each complete input/frame is bounded; slow HTTP headers/bodies get a ten-second read deadline. HTTP write deadline is at least the maximum review duration plus cleanup/delivery allowance (310 seconds), and idle keep-alive is 30 seconds. Active review capacity is acquired only after validation; no background queue stores material. A bounded transport admission guard permits at most four concurrently decoded messages per process and rejects excess HTTP work before reading a body; stdio rejects excess tool work immediately and keeps reading bounded frames so cancellation notifications cannot be starved. It must not enqueue blocked handlers or pause the sole reader behind long-running reviews. These transport slots are not review jobs or durable state.

Capping only the final MCP response would leave the current provider buffers unbounded. Therefore implementation must add optional execution capture limits to provider construction (outside the content-only `provider.Request`), shared by all subprocess adapters, with a distinct `ErrOutputLimit` sentinel. MCP always sets these limits; existing CLI behavior remains unchanged unless separately opted in. The helper cancels execution on overflow and joins readers before returning. It does not silently discard excess bytes and then report success. Future HTTP adapters must honor the same output-budget semantics.

### D7. Layering

```text
stdio / stateless HTTP
        │
        ▼
internal/mcpserver
  argument validation, tool schemas, admission,
  error mapping, result encoding
        │
        ▼
internal/review executor
  resolve prompt, construct provider per call,
  result validation without output formatting
        │
        ▼
internal/provider
  Request{Prompt, Material} → Result{Findings, Provenance}
  optional output limits; existing cold adapters
```

Use a small shared executor returning the current provider result; keep MCP DTOs and receipt calculation out of provider adapters. Receipt hashes are computed by the MCP adapter from its resolved request. The CLI may adopt prompt/provider orchestration through this executor only with behavior-preserving tests; it keeps its own file/diff assembly and output/exit semantics. Never invoke the CLI as a subprocess from MCP or parse its stderr. Do not share mutable provider instances across calls.

The registry stays authoritative. `list_providers` reads it; it does not advertise Loopback or providers not implemented. Tool descriptions explicitly say material can leave the machine, model independence is caller-selected, and default markers do not establish model identity. Review annotations are conservative: `readOnlyHint=false`, `destructiveHint=false`, `idempotentHint=false`, `openWorldHint=true`; external work/cost and provider side effects prevent a blanket read-only/idempotent claim. Discovery is read-only, idempotent, and closed-world.

### D8. Relationship to findings-envelope

The pending proposal assumes parsed findings block MCP. This design supersedes that dependency only: raw text plus provenance is sufficient. It does not implement, reject, or rename that proposal's parser types. An optional future parsed view must preserve `raw`, disclose parse status, and get its own schema/version decision. No novelty score, ranking, author-model inference, or same-model enforcement is added here.

## Cold-reviewer guarantee

The gateway never dereferences paths, reads a repository, asks for roots/history, samples the calling model, or injects MCP client metadata into material. All content passed to `Provider.Review` comes from the tool arguments and explicit baked prompt selection. Deployment authentication and isolation settings are not review context. Existing provider limitations remain visible; adding an MCP wrapper is not new evidence of provider isolation.

## Testing and acceptance

- Pure argument cases: absent versus empty/null prompt, UTF-8/escape validation, multibyte size boundaries, unknown fields/provider, model and timeout limits. Compare exact request bytes at the engine seam.
- Provider-result cases: empty raw, full provenance preservation, hashes from resolved prompt/material, Unicode and whitespace, invalid result, sentinel-to-error mapping, structured/text equivalence, maximum encoded size.
- Statelessness: A then B with different inputs; B omits optional inputs and receives defaults rather than A's values. Fresh server produces the same request assembly. Repeated JSON-RPC IDs on separate transports never reuse results. No calls to filesystem assembly helpers or client roots/sampling.
- SDK client against actual in-memory/stdio/loopback HTTP transports, both supported protocol versions. Assert stdout purity, stateless HTTP behavior, no session cookies/IDs, correct framing, token/Origin/Host rejection, no token inheritance by child environments, body/frame limits, and no event-store dependence.
- Start two requests, reject a third as busy, cancel one and verify the other survives and the slot is reused. Saturate transport admission separately. Test timeout and shutdown with cooperative Loopback-derived test providers at the engine seam, not fake provider executables.
- Real subprocess tests for stdout/stderr overflow cancel/cleanup and no partial success. Race tests and memory-bounded stress tests for rejected/saturated calls.
- Integration-tagged real-provider positive control and cold conformance remain required. One real MCP review per verified adapter demonstrates the gateway supplies material and preserves provenance. Claude's current OAuth blocker must be recorded rather than hidden behind Loopback success.

## Risks / Trade-offs

Statelessness gives no recovery or exactly-once execution after a lost response. Fixed budgets may reject useful large reviews; explicit errors are preferable to silent truncation. Duplicated JSON text increases wire size but supports text-only clients. HTTP adds local credential handling; stdio remains simplest. Global-context isolation and detached subprocess limits are inherited, not solved here. Legacy compatibility adds protocol bookkeeping but no review history.

## Migration Plan

Implement behind a new subcommand, leaving existing invocations unchanged. Add harness configuration examples and update the canonical skill. Keep current implementation specs unchanged until apply; after passing acceptance tests, sync this delta and update the MCP milestone with evidence. Rollback removes only the new subcommand and dependency plus unused bounded-output options; no stored data needs migration.

## Sources checked 2026-11-13

- [Official Go SDK lifecycle/transport documentation](https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/protocol.md): modern per-request metadata, legacy compatibility, stateless HTTP option.
- [MCP tools specification](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/server/tools.mdx): structured results, schemas, and tool/protocol error distinction.
- [MCP stdio transport](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2026-07-28/basic/transports/stdio.mdx): message framing, stdout discipline, cancellation.
- [Official Go SDK releases](https://github.com/modelcontextprotocol/go-sdk/releases): pin a tested stable release during implementation; repository currently has no SDK dependency.
