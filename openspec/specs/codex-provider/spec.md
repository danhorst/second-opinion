# codex-provider

## Purpose

The codex adapter's provider-specific requirements: cold invocation, JSONL findings extraction, model provenance under an unstructured harness, auth-aware model fallback, did-not-run classification.

## Requirements

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

### Requirement: Provenance under an unstructured harness
Codex's `--json` event stream carries no model field, on forced or unforced runs alike. When a model is forced, provenance SHALL name it; when none is forced, provenance SHALL carry the documented `codex-default` marker rather than a guess.

#### Scenario: Forced model
- **WHEN** a review runs with a forced model
- **THEN** the result's provenance names that model

#### Scenario: Default model
- **WHEN** a review runs without a forced model
- **THEN** the result's provenance reads `codex-default`, a documented degradation, not an invented model name

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

### Requirement: Validate completed transport output
The adapter SHALL return a result only for well-formed JSONL with a recognized agent message followed by successful terminal completion. Explicit empty message strings SHALL be valid. It SHALL reject malformed, oversized, incomplete, and failed streams and SHALL NOT search arbitrary nested objects for findings.

#### Scenario: Completed empty findings
- **WHEN** an empty agent message is followed by successful completion
- **THEN** the adapter returns empty findings and provenance

#### Scenario: Invalid or incomplete stream
- **WHEN** parsing fails, a turn fails, or successful completion or a recognized message is missing
- **THEN** the adapter returns an error and no findings result

#### Scenario: Nonfatal item warning
- **WHEN** a well-formed warning item precedes a completed review
- **THEN** the warning does not invalidate the successful review

### Requirement: Suppress optional context and tools
Codex invocations SHALL ignore user configuration and execpolicy rules, suppress host skill discovery, and disable shell execution, code-mode host, local image reading, apps, plugins, memory, multi-agent, skill search, and web search. Authentication SHALL remain available. Unsupported controls SHALL fail rather than retry with weaker isolation.

#### Scenario: User customizations are present
- **WHEN** an invocation is constructed
- **THEN** explicit context/tool suppression controls are present in addition to isolated cwd and project-document suppression
