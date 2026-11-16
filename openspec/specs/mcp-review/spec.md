# mcp-review

## Purpose

Provide self-contained adversarial reviews through MCP without retaining review history or allowing implicit workspace access.

## Requirements

### Requirement: Stateless review application
Every review SHALL be determined by its complete tool arguments, the selected baked prompt when omitted, and immutable deployment/provider configuration. The server SHALL NOT retain conversations, material, findings, result handles, job records, caches, or deduplication state after execution. Only bounded in-flight request/cancellation/transport bookkeeping SHALL exist. Request identifiers SHALL NOT be idempotency keys.

#### Scenario: Prior call cannot supply context
- **WHEN** a client performs call A and then call B with optional model and prompt omitted
- **THEN** B uses defaults and receives no model, prompt, material, or findings from A

#### Scenario: Server restarts
- **WHEN** the process restarts after an interrupted review
- **THEN** no review can be resumed or retrieved; another call is an independent execution

### Requirement: Explicit content and provider selection
The `review` tool SHALL require a registered `provider` and complete UTF-8 `material`, and SHALL accept only optional `model`, `prompt`, and `timeout_ms` in addition. Missing prompt SHALL select the baked prompt; a supplied prompt SHALL replace it entirely. Missing model SHALL select the provider default. MCP SHALL NOT default its provider from environment variables, client identity, or previous requests. Unknown fields, null/wrong types, blank required text, and invalid values SHALL fail before provider invocation. Accepted text SHALL remain unmodified.

#### Scenario: Caller supplies a patch
- **WHEN** material contains a caller-generated diff and deliberately included supporting text
- **THEN** those exact bytes are passed with the resolved prompt to the selected provider

#### Scenario: Material references a file or URL
- **WHEN** material names a path, repository, or URL without its contents
- **THEN** the MCP server does not dereference it or fetch additional context

#### Scenario: Invalid explicit selection
- **WHEN** provider is absent or unknown, or an argument is invalid
- **THEN** no provider call is started and an input error is returned

### Requirement: Registry-only discovery
The `list_providers` tool SHALL accept an empty object and return schema version 1, kind `providers`, sorted registered provider names, and the effective limits. It SHALL NOT inspect authentication, probe executables or endpoints, or invoke a model. Tool discovery SHALL be deterministic and SHALL NOT vary because of previous review requests.

#### Scenario: Provider is unavailable
- **WHEN** a registered provider's binary is missing
- **THEN** it remains listed and availability is checked only when explicitly selected for review

### Requirement: Lossless versioned review outcomes
A successful review SHALL return schema version 1, kind `review`, raw provider-returned text, unchanged provider/model/full-prompt-hash provenance, and a request receipt with SHA-256 material hash, material byte length, and effective timeout. It SHALL NOT parse, count, rank, or infer novelty of findings. Default-model markers SHALL remain unchanged and SHALL NOT be represented as verified independence from the author.

Each tool outcome SHALL appear in structured content and exactly one text block containing the equivalent serialized JSON value. A declared output schema SHALL cover the tool's success and failure variants. Protocol framing SHALL be produced for the supported protocol version.

#### Scenario: Valid empty findings
- **WHEN** an adapter returns a validated successful result with empty text
- **THEN** the tool returns success with empty raw text and complete provenance

#### Scenario: Text-only caller
- **WHEN** a caller reads only the tool's text block
- **THEN** it can recover the same raw text, provenance, and receipt as structured-content callers

### Requirement: Actionable failures without false completion
Tool validation and execution errors SHALL set `isError=true` and return schema version 1, kind `error`, and an error object with `code`, safe `message`, and `execution` (`not_started` or `unknown`). Codes SHALL include `invalid_input`, `input_too_large`, `busy`, `unavailable`, `auth`, `timeout`, `cancelled`, `output_too_large`, and `provider_failed`. Failures SHALL NOT contain a fabricated successful review or provenance. Malformed protocol requests and unknown methods/tools SHALL remain protocol errors.

Provider diagnostics, material, credentials, and output fragments SHALL NOT be copied into error messages. The gateway SHALL NOT automatically retry a review or promise exactly-once execution. Existing provider model fallback remains permitted within the same deadline.

#### Scenario: Deadline expires
- **WHEN** the server deadline expires while the connection remains writable
- **THEN** it returns a timeout tool error with execution unknown and no findings result

#### Scenario: Client retries a lost response
- **WHEN** a caller resends material after losing a response
- **THEN** it triggers an independent review with no cached result or deduplication guarantee

### Requirement: Bounded execution and admission
The server SHALL enforce: 8 MiB transport requests; 1 MiB combined material/resolved prompt; 256-byte model names; 120000 ms default and 300000 ms maximum timeout; two simultaneous reviews; four in-flight transport messages; 8 MiB provider stdout and 64 KiB provider stderr per invocation; 1 MiB raw findings; and 16 MiB encoded responses. Text limits SHALL count UTF-8 bytes. Lossy decoding SHALL be rejected. Limits SHALL be advertised by discovery and SHALL NOT be expanded by tool input.

An unavailable review slot SHALL return `busy` before provider invocation, without storing a pending review. Provider capture overflow SHALL cancel execution and fail without partial results. The stdio reader SHALL continue accepting cancellation notifications under saturation, rather than waiting behind running reviews. HTTP SHALL reject excess admission before reading a body and bound slow header/body reads to ten seconds.

#### Scenario: Multibyte input exceeds byte budget
- **WHEN** accepted Unicode characters exceed the combined UTF-8 input limit
- **THEN** the tool returns input-too-large before invoking a provider

#### Scenario: A third review arrives
- **WHEN** two reviews are executing
- **THEN** another review returns busy without entering a background queue or invoking a provider

#### Scenario: Provider output exceeds its capture limit
- **WHEN** stdout or stderr exceeds its per-invocation cap
- **THEN** the invocation is cancelled and output-too-large is returned without partial findings

### Requirement: Cancellation and shutdown
Every review SHALL use the earliest of its requested/default deadline, client cancellation/disconnection, and server shutdown. Cancellation SHALL reach the provider and release admission on completion. One request's cancellation SHALL NOT cancel another. After explicit client cancellation or disconnection, response delivery SHALL NOT be promised; version-specific rules prohibiting further messages SHALL be obeyed. EOF or SIGTERM SHALL cancel outstanding work and bound shutdown waiting to five seconds.

#### Scenario: One client cancels
- **WHEN** one of two active reviews is cancelled
- **THEN** its provider receives cancellation, its slot is released after cleanup, and the other review continues

#### Scenario: Client disconnects
- **WHEN** the originating transport closes
- **THEN** associated work is cancelled and no result is stored for later retrieval

### Requirement: Stdio and stateless HTTP serving
`second-opinion mcp serve` SHALL default to stdio; optional `--transport http --listen ADDRESS` SHALL serve `/mcp` using stateless Streamable HTTP without persisted session identifiers, event stores, resumability, or a persistent server-event GET stream. Stdio stdout SHALL contain only MCP messages. Both transports SHALL support protocol 2026-07-28 and explicitly tested 2025-11-25 compatibility without review state. No application semantics SHALL depend on a transport session.

#### Scenario: Stdio tool failure
- **WHEN** a valid tool call fails
- **THEN** a tool result is returned through MCP and the server remains available for further calls

#### Scenario: HTTP request lands on a fresh process
- **WHEN** a request includes the required authentication, protocol metadata, and complete tool input
- **THEN** it can execute without prior review requests or stored session state

### Requirement: Private local HTTP boundary
HTTP SHALL bind only literal loopback addresses, require a deployment bearer token of at least 32 bytes from `SECOND_OPINION_MCP_TOKEN`, authenticate each request before reading tool input, reject browser Origin headers and unexpected Host authorities, and provide no CORS or public-binding bypass. The token SHALL NOT appear in logs, arguments, or provider child environments. Stdio SHALL rely on OS process access and require no gateway token. Provider credentials SHALL remain server-owned and SHALL NOT be accepted in review arguments.

#### Scenario: Unauthorized or browser request
- **WHEN** an HTTP request has missing/invalid authorization, an Origin header, or an unexpected Host
- **THEN** it is rejected before provider invocation without revealing input or credentials

#### Scenario: Non-loopback listen address
- **WHEN** startup requests a wildcard, hostname, or non-loopback bind
- **THEN** the server fails startup instead of exposing authenticated provider access publicly

### Requirement: Cold transport-neutral engine boundary
The MCP implementation SHALL call the shared review engine/provider interface directly and SHALL NOT shell out to the second-opinion CLI. Client identity, roots, history, tool metadata, and previous findings SHALL NOT be added to provider content. The server SHALL expose no sampling, elicitation, file-reading, resource subscription, or result retrieval features. Provider isolation limitations SHALL remain documented rather than being represented as solved by MCP.

#### Scenario: Caller advertises extra capabilities
- **WHEN** client capabilities include roots or sampling
- **THEN** the gateway does not request or include them in a review

### Requirement: Material provenance across front-ends
MCP and CLI JSON success provenance SHALL include the full material_hash supplied by the provider. The shared executor SHALL verify it against the exact input material. The existing MCP request receipt SHALL retain its material hash and remain consistent with provenance.

#### Scenario: Reconciled execution
- **WHEN** a request uses the shared executor and configured output budgets
- **THEN** input material identity is preserved and excessive output fails explicitly
