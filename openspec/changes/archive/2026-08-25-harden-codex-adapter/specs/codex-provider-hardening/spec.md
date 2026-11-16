# codex-provider-hardening

## ADDED Requirements

### Requirement: Codex suppresses ambient review context
The Codex adapter SHALL invoke `codex exec` from a fresh temporary directory with project-document loading disabled, user configuration ignored, execution-policy rules ignored, a read-only sandbox, repository checks skipped, and session persistence disabled.
The adapter SHALL document this as prevention of implicit context, not as strict prevention of deliberate filesystem reads.

#### Scenario: Invocation excludes ambient configuration
- **WHEN** a review is executed
- **THEN** the invocation includes the empty temporary working directory and all supported ambient-context suppression flags

### Requirement: Codex success requires recognized protocol data
The adapter SHALL return a successful result only when the JSONL stream contains a recognized agent message and sufficient model provenance.
A zero-exit process with an empty, malformed, or otherwise unrecognized event stream SHALL return a did-not-run protocol error.
When Codex omits model identity on an unforced run, the adapter SHALL report the explicit `codex-default` provenance marker.

#### Scenario: Unknown success stream fails closed
- **WHEN** Codex exits successfully without a recognized agent message
- **THEN** the adapter returns an error and no result

#### Scenario: Valid empty findings remain valid
- **WHEN** Codex emits a recognized agent message whose text is empty
- **THEN** the adapter returns a successful result with empty findings

#### Scenario: Default model identity is unavailable
- **WHEN** Codex completes an unforced review without emitting a model identity
- **THEN** the adapter returns a successful result with `codex-default` as the documented model marker

### Requirement: Codex reports material identity
The Codex adapter SHALL populate result provenance with the SHA-256 identity of the exact material passed on stdin.

#### Scenario: Stdin material is auditable
- **WHEN** a review completes
- **THEN** `Provenance.MaterialHash` equals the shared material hash of `Request.Material`

### Requirement: Codex fallback is narrow and cancellation-safe
When a forced model is rejected by the known ChatGPT-account model-authentication signature, the adapter SHALL retry exactly once without a forced model only if the request context remains active.
Any other failure SHALL NOT trigger fallback.

#### Scenario: Cancellation does not retry
- **WHEN** the first invocation is cancelled
- **THEN** the adapter returns the cancellation error without starting a default-model retry

#### Scenario: Nested model rejection triggers fallback
- **WHEN** a forced-model invocation fails with the model-rejection signature nested in a Codex failure event
- **THEN** the adapter retries once without `-m` and reports the model from the successful response

### Requirement: Codex failures preserve classification
The adapter SHALL extract failure messages from both top-level and nested JSONL error events and SHALL classify missing binaries as unavailable, usage exhaustion as unavailable, and authentication rejection as authentication failure while preserving diagnostic detail.

#### Scenario: Nested usage limit is unavailable
- **WHEN** Codex reports a usage limit in a nested failure event
- **THEN** the adapter returns an error matching `ErrUnavailable`

### Requirement: Codex behavior is verified against the real binary
The adapter SHALL retain deterministic unit coverage for invocation and parsing and SHALL run the shared conformance and provenance tests against the real Codex binary behind the `integration` build tag.

#### Scenario: Installed Codex event schema is verified
- **WHEN** the integration suite runs with an authenticated Codex binary
- **THEN** the conformance suite passes and provenance names the model that actually ran
