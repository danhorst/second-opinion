# provider-interface

## Purpose

The contract between the review engine and any reviewer: request shape, result shape with provenance, error semantics, transport-agnosticism.

## Requirements

### Requirement: One interface for all reviewers
The system SHALL define a single `Provider` interface that every reviewer adapter implements, regardless of transport.
The interface MUST NOT expose transport details — a caller cannot tell from the interface whether a provider shells out to a binary or speaks HTTP.

#### Scenario: Subprocess and HTTP adapters satisfy the same interface
- **WHEN** a subprocess-backed adapter and an HTTP-backed adapter are both compiled against the `Provider` interface
- **THEN** both satisfy it without transport-specific methods, parameters, or type assertions at the call site

### Requirement: Review request carries explicitly assembled material only
A review request SHALL consist of exactly two content inputs: the review prompt and the material to review.
The material MUST be fully assembled by the caller before the request is made; the interface SHALL offer no way to pass a path, repository reference, or other pointer the provider would dereference itself.

#### Scenario: Material is passed by value
- **WHEN** a caller constructs a review request
- **THEN** the request holds the material's content, not a location to fetch it from

#### Scenario: Auditable material
- **WHEN** a review request is executed
- **THEN** it is possible to reproduce byte-for-byte what the reviewer received

### Requirement: Review result carries reviewer provenance
Every review result SHALL include provenance identifying the provider, the model that performed the review, the identity of the prompt used, and the identity of the material reviewed.
Provenance MUST survive to the caller unaltered so that a reviewer-equals-author violation and a material mismatch are detectable after the fact.

#### Scenario: Provenance on a successful review
- **WHEN** a provider completes a review
- **THEN** the result names the provider, the model that ran, the prompt identity, and the SHA-256 identity of the exact material value received by the provider

#### Scenario: Reported model reflects what actually ran
- **WHEN** a provider substitutes a different model than requested (for any provider-internal reason)
- **THEN** the result's provenance names the model that actually performed the review

#### Scenario: Material identity is transport-independent
- **WHEN** two providers receive byte-identical material
- **THEN** both results report the same material identity regardless of whether the providers use a subprocess, HTTP, or another transport

### Requirement: Ran and did-not-run are distinguishable
The interface's error semantics SHALL distinguish a reviewer that ran (its findings, even if empty, are in the result) from a reviewer that did not run (an error, no result).
A result and an error MUST be mutually exclusive.

#### Scenario: Reviewer runs and finds nothing
- **WHEN** the reviewer completes and reports no findings
- **THEN** the caller receives a result with empty findings and no error

#### Scenario: Reviewer fails to run
- **WHEN** the provider cannot execute the review (binary missing, endpoint unreachable, auth rejected)
- **THEN** the caller receives an error identifying the failure and no result

### Requirement: Requests are cancellable
Every review SHALL accept a context, and a provider MUST abandon work and return promptly when that context is cancelled.

#### Scenario: Caller cancels a slow review
- **WHEN** the caller cancels the context while a review is in flight
- **THEN** the provider returns a cancellation error without waiting for the reviewer to finish

### Requirement: Subprocess cleanup and bounded output drain
Subprocess adapters SHALL terminate their invocation process group on cancellation and completion on Darwin, Linux, and BSD. They SHALL bound waits for inherited output pipes to two seconds after cancellation or parent exit. On other platforms they SHALL document parent-only cancellation. Detached descendants outside the invocation group are not covered.

#### Scenario: Child holds output open
- **WHEN** a provider parent exits or is cancelled while descendants retain output pipes
- **THEN** the caller does not wait indefinitely for those pipes and supported-platform group members are terminated

### Requirement: Actionable argument-size errors
When an OS rejects provider arguments as too large, the adapter SHALL explain that material must be reduced without including material or secrets in the error, and SHALL preserve the underlying error.

#### Scenario: Oversized command arguments
- **WHEN** starting an argument-based provider fails with the OS argument-limit error
- **THEN** the result is an error with actionable size guidance and no completed review

### Requirement: Optional execution output budgets
Provider construction SHALL support optional stdout/stderr capture budgets without adding non-content fields to the review request. When configured, every subprocess invocation including fallback SHALL enforce the budgets, cancel work on overflow, and return a distinguishable output-limit error without a completed or truncated result. Omitted limits SHALL preserve existing CLI behavior.

#### Scenario: Budgeted invocation overflows
- **WHEN** provider stdout or stderr exceeds its configured capture budget
- **THEN** execution is cancelled, readers are cleaned up, and an output-limit error is returned without findings

#### Scenario: Existing CLI caller
- **WHEN** a provider is constructed without execution output budgets
- **THEN** existing CLI output behavior is unchanged

### Requirement: HTTP response budgets
HTTP providers SHALL honor configured output budgets for success and error response bodies and return the output-limit sentinel without partial success when exceeded. Success body limits SHALL be separate from diagnostic truncation limits.

#### Scenario: Reconciled execution
- **WHEN** a request uses the shared executor and configured output budgets
- **THEN** input material identity is preserved and excessive output fails explicitly

### Requirement: Optional isolation execution policy
Callers SHALL be able to require `local-material-only` separately from the content-only review request. A provider lacking support SHALL return a distinguishable isolation-unsupported error before Review execution, with no result or fallback. Unknown requirements SHALL fail closed. An isolation error returned after provider invocation SHALL NOT be classified as a pre-execution rejection. Successful results SHALL carry the effective local boundary separately from reviewer provenance.

#### Scenario: Required boundary unsupported
- **WHEN** a caller requires local-material-only from an unsupported provider
- **THEN** the reviewer is not invoked and the caller receives the isolation-unsupported error

#### Scenario: Requirement omitted
- **WHEN** no isolation requirement is supplied
- **THEN** existing provider selection remains available and the result identifies its effective boundary
