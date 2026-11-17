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
