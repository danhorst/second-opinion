## ADDED Requirements

### Requirement: MCP isolation controls and receipts
The review tool SHALL accept optional `require_isolation` with only `local-material-only` as a valid value. Unknown, null, and non-string values SHALL fail validation. Unsupported execution SHALL return `isolation_unsupported` with execution `not_started`. Successful output SHALL include effective `isolation`; list_providers SHALL include `provider_isolation` mapping each registered provider to its effective boundary. Schemas SHALL describe the local scope and endpoint trust.

#### Scenario: Unsupported MCP review
- **WHEN** a valid local-material-only requirement targets an unsupported provider
- **THEN** the tool returns isolation_unsupported and not_started without invoking Review

#### Scenario: Invalid isolation value
- **WHEN** require_isolation is null or unknown
- **THEN** the tool returns invalid_input and not_started

## MODIFIED Requirements

### Requirement: Explicit content and provider selection
The `review` tool SHALL require a registered `provider` and complete UTF-8 `material`, and SHALL accept only optional `model`, `prompt`, `timeout_ms`, and `require_isolation` in addition. Missing prompt SHALL select the baked prompt; a supplied prompt SHALL replace it entirely. Missing model SHALL select the provider default. MCP SHALL NOT default its provider from environment variables, client identity, or previous requests. Unknown fields, null/wrong types, blank required text, and invalid values SHALL fail before provider invocation. Accepted text SHALL remain unmodified.

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
The `list_providers` tool SHALL accept an empty object and return schema version 2, kind `providers`, sorted registered provider names, and the effective limits. It SHALL NOT inspect authentication, probe executables or endpoints, or invoke a model. Tool discovery SHALL be deterministic and SHALL NOT vary because of previous review requests.

#### Scenario: Provider is unavailable
- **WHEN** a registered provider's binary is missing
- **THEN** it remains listed and availability is checked only when explicitly selected for review

### Requirement: Lossless versioned review outcomes
A successful review SHALL return schema version 2, kind `review`, raw provider-returned text, unchanged provider/model/full-prompt-hash provenance, and a request receipt with SHA-256 material hash, material byte length, and effective timeout. It SHALL NOT parse, count, rank, or infer novelty of findings. Default-model markers SHALL remain unchanged and SHALL NOT be represented as verified independence from the author.

Each tool outcome SHALL appear in structured content and exactly one text block containing the equivalent serialized JSON value. A declared output schema SHALL cover the tool's success and failure variants. Protocol framing SHALL be produced for the supported protocol version.

#### Scenario: Valid empty findings
- **WHEN** an adapter returns a validated successful result with empty text
- **THEN** the tool returns success with empty raw text and complete provenance

#### Scenario: Text-only caller
- **WHEN** a caller reads only the tool's text block
- **THEN** it can recover the same raw text, provenance, and receipt as structured-content callers

### Requirement: Actionable failures without false completion
Tool validation and execution errors SHALL set `isError=true` and return schema version 2, kind `error`, and an error object with `code`, safe `message`, and `execution` (`not_started` or `unknown`). Codes SHALL include `invalid_input`, `input_too_large`, `busy`, `unavailable`, `auth`, `timeout`, `cancelled`, `output_too_large`, and `provider_failed`. Failures SHALL NOT contain a fabricated successful review or provenance. Malformed protocol requests and unknown methods/tools SHALL remain protocol errors.

Provider diagnostics, material, credentials, and output fragments SHALL NOT be copied into error messages. The gateway SHALL NOT automatically retry a review or promise exactly-once execution. Existing provider model fallback remains permitted within the same deadline.

#### Scenario: Deadline expires
- **WHEN** the server deadline expires while the connection remains writable
- **THEN** it returns a timeout tool error with execution unknown and no findings result

#### Scenario: Client retries a lost response
- **WHEN** a caller resends material after losing a response
- **THEN** it triggers an independent review with no cached result or deduplication guarantee
