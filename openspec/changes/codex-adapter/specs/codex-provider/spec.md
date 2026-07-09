# codex-provider

## ADDED Requirements

### Requirement: Codex runs cold by construction
The codex adapter SHALL invoke `codex exec` from an empty temporary directory with project-document loading suppressed and a read-only, ephemeral sandbox.
The invocation MUST NOT inherit the caller's working directory.

#### Scenario: Invocation is isolated from the caller's repository
- **WHEN** the adapter runs a review while the caller's working directory contains instruction files and repository content
- **THEN** the codex process runs from an empty temporary directory with project-doc loading disabled, and the review passes the conformance suite's cold checks

### Requirement: Material travels on stdin
The adapter SHALL pass the prompt as the codex instruction and the material on stdin, so codex receives the material as an appended block rather than a file path to dereference.

#### Scenario: Review of assembled material
- **WHEN** the adapter executes a review request
- **THEN** the material reaches codex verbatim via stdin and no temporary file of the material is left behind

### Requirement: Provenance reflects the codex session
The adapter SHALL extract the model that actually ran from codex's structured event stream and report it in the result's provenance.

#### Scenario: Default model run
- **WHEN** a review runs without a forced model
- **THEN** provenance names the model codex actually selected, not an empty string or a guess

### Requirement: Auth-aware model fallback
When a forced model is rejected because of the active codex authentication (ChatGPT-account auth rejects some models), the adapter SHALL retry once on the provider default and report the substituted model in provenance.
Any other failure SHALL NOT trigger the fallback.

#### Scenario: Forced model unavailable under active auth
- **WHEN** a review is requested with a forced model that codex rejects as unsupported for the active account type
- **THEN** the adapter retries on the default model and the result's provenance names the model that actually ran

#### Scenario: Unrelated failure is not retried
- **WHEN** codex fails for a reason other than model-auth rejection
- **THEN** the adapter returns a did-not-run error without retrying

### Requirement: Did-not-run classification
The adapter SHALL map failure modes onto the contract's sentinel errors: a missing `codex` binary yields `ErrUnavailable`, an authentication failure yields `ErrAuth`, and both preserve the underlying detail.

#### Scenario: Binary missing
- **WHEN** `codex` is not on PATH
- **THEN** the adapter returns an error matching `ErrUnavailable` and no result

### Requirement: Conformance behind the integration tag
The adapter SHALL pass the shared conformance suite against the real `codex` binary, gated behind the `integration` build tag, and SHALL add deterministic white-box assertions that the constructed invocation carries the cold flags.

#### Scenario: White-box cold flags
- **WHEN** the adapter constructs a codex invocation
- **THEN** a unit test can assert it contains the empty-tempdir working directory, project-doc suppression, and the read-only ephemeral sandbox flags without executing codex

#### Scenario: Real-binary conformance
- **WHEN** `make test-integration` runs with `codex` on PATH
- **THEN** the adapter passes the full conformance suite against the real binary
