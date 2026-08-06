# provider-conformance

## ADDED Requirements

### Requirement: Conformance proves the material reaches the reviewer
The suite SHALL include a positive control: it plants a distinctive canary token in the material and asserts the reviewer's output reflects that token.
An adapter that discards, truncates, or fails to transmit the material MUST fail this check, even when the reviewer returns confident, non-empty output.

#### Scenario: Material reaches the reviewer
- **WHEN** the suite runs a review whose material contains a canary token and instructs the reviewer to repeat it
- **THEN** the reviewer's output contains that token

#### Scenario: Adapter that sends no material fails
- **WHEN** the suite runs against a reference provider rigged to discard the request's material and return unrelated non-empty output
- **THEN** the positive control fails

#### Scenario: Non-emptiness is not sufficient
- **WHEN** a provider returns output that is non-empty but unrelated to the material
- **THEN** the positive control fails, because output length and presence are not evidence of transmission

## MODIFIED Requirements

### Requirement: Conformance proves the cold-reviewer guarantee
The suite SHALL verify that the reviewer receives nothing the caller did not explicitly assemble into the request — no project instruction files auto-loaded from the working directory, no access to the caller's repository, no session history.
These checks SHALL be reported as meaningful only when the positive control passes: a reviewer that received no material trivially satisfies every negative assertion, so a cold result without a passing positive control MUST NOT be reported as a pass.

#### Scenario: Reviewer cannot see project instruction files
- **WHEN** the suite runs a review from a directory containing an instruction file the provider's tooling would auto-load (for example `AGENTS.md`)
- **THEN** the reviewer's context provably excludes that file's content

#### Scenario: Reviewer cannot read the caller's repository
- **WHEN** the suite runs a review of material referencing files that exist in the caller's repository but were not included in the material
- **THEN** the reviewer provably cannot read those files

#### Scenario: Cold results are not reported beneath a failing positive control
- **WHEN** the positive control fails for an adapter
- **THEN** the suite reports that the cold checks are inconclusive rather than passing, so a milestone cannot cite a green result
