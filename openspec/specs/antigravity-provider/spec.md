# antigravity-provider

## Purpose

The antigravity (`agy`) adapter's provider-specific requirements: cold invocation, prompt-embedded material, model forcing, did-not-run classification.

## Requirements

### Requirement: Antigravity runs cold by construction
The antigravity adapter SHALL invoke `agy --print` with its process working directory set to an empty temporary directory, sandbox restrictions enabled, and no workspace directories granted.
The invocation MUST NOT inherit the caller's working directory.

#### Scenario: Invocation is isolated from the caller's repository
- **WHEN** the adapter runs a review while the caller's working directory contains instruction files and repository content
- **THEN** the agy process runs from an empty temporary directory and the review passes the conformance suite's cold checks

### Requirement: Material is embedded in the prompt
Because `agy --print` does not read stdin, the adapter SHALL compose a single instruction from the review prompt and the material, with the material in a clearly delimited block.
The composition MUST be a pure function so the exact instruction the reviewer received is reproducible.

#### Scenario: Composed instruction is auditable
- **WHEN** the adapter executes a review request
- **THEN** the instruction handed to agy is a deterministic function of the request's prompt and material, and contains the material verbatim

### Requirement: Provenance under an unstructured harness
When a model is forced, provenance SHALL name it; when none is forced, provenance SHALL carry the documented `antigravity-default` marker rather than a guess.
The provider name SHALL be `antigravity`.

#### Scenario: Forced model
- **WHEN** a review runs with a forced model
- **THEN** the result's provenance names that model

#### Scenario: Default model
- **WHEN** a review runs without a forced model
- **THEN** the result's provenance reads `antigravity-default`, a documented degradation, not an invented model name

### Requirement: Did-not-run classification
The adapter SHALL map failure modes onto the contract's sentinel errors: a missing `agy` binary yields `ErrUnavailable`, an authentication or sign-in failure yields `ErrAuth`, and both preserve the underlying detail.

#### Scenario: Binary missing
- **WHEN** `agy` is not on PATH
- **THEN** the adapter returns an error matching `ErrUnavailable` and no result

### Requirement: Conformance behind the integration tag
The adapter SHALL pass the shared conformance suite against the real `agy` binary, gated behind the `integration` build tag, with deterministic white-box assertions on the constructed invocation.

#### Scenario: White-box cold invocation
- **WHEN** the adapter constructs an agy invocation
- **THEN** a unit test can assert the sandbox flag, print mode, the composed instruction, and the absence of workspace grants without executing agy

#### Scenario: Real-binary conformance
- **WHEN** `make test-integration` runs with `agy` on PATH and an authenticated session
- **THEN** the adapter passes the full conformance suite against the real binary
