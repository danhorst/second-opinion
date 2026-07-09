# provider-interface

## ADDED Requirements

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
Every review result SHALL include provenance identifying the provider, the model that performed the review, and the identity of the prompt used.
Provenance MUST survive to the caller unaltered so that a reviewer-equals-author violation is detectable after the fact.

#### Scenario: Provenance on a successful review
- **WHEN** a provider completes a review
- **THEN** the result names the provider, the model that ran, and the prompt identity

#### Scenario: Reported model reflects what actually ran
- **WHEN** a provider substitutes a different model than requested (for any provider-internal reason)
- **THEN** the result's provenance names the model that actually performed the review

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
