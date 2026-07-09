# provider-conformance

## Purpose

The shared test suite that proves an adapter honors the provider contract, above all the cold-reviewer guarantee.

## Requirements

### Requirement: Shared conformance suite for all adapters
The system SHALL provide one conformance suite that any `Provider` implementation can be run through, and every adapter MUST pass it before it is considered complete.
The suite SHALL be the same code for every adapter — an adapter cannot pass a private variant.

#### Scenario: New adapter adopts the suite
- **WHEN** a new adapter package is added
- **THEN** it passes by invoking the shared suite against its own constructor, with no adapter-specific assertions about the contract

### Requirement: Conformance proves the cold-reviewer guarantee
The suite SHALL verify that the reviewer receives nothing the caller did not explicitly assemble into the request — no project instruction files auto-loaded from the working directory, no access to the caller's repository, no session history.

#### Scenario: Reviewer cannot see project instruction files
- **WHEN** the suite runs a review from a directory containing an instruction file the provider's tooling would auto-load (for example `AGENTS.md`)
- **THEN** the reviewer's context provably excludes that file's content

#### Scenario: Reviewer cannot read the caller's repository
- **WHEN** the suite runs a review of material referencing files that exist in the caller's repository but were not included in the material
- **THEN** the reviewer provably cannot read those files

### Requirement: Conformance verifies the interface contract
The suite SHALL exercise the contract requirements of `provider-interface`: provenance present and truthful, ran/did-not-run mutually exclusive, and cancellation honored.

#### Scenario: Contract checks run against every adapter
- **WHEN** the suite runs against an adapter
- **THEN** it asserts provenance fields are populated, an empty review yields a result and not an error, an unreachable reviewer yields an error and not a result, and a cancelled context returns promptly

### Requirement: Conformance runs against the real reviewer
The suite SHALL exercise each adapter against its real binary or endpoint, gated behind a build tag so the default `make test` run does not require external tools or make network calls.
The provider boundary MUST NOT be mocked to satisfy the suite.

#### Scenario: Default test run stays hermetic
- **WHEN** `make test` runs without the integration build tag
- **THEN** no external binary is executed and no network call is made

#### Scenario: Tagged run uses the real reviewer
- **WHEN** the suite runs with the integration build tag
- **THEN** each adapter under test invokes its actual binary or endpoint

### Requirement: The suite itself is validated
The system SHALL include an in-process reference provider whose only purpose is to exercise the suite, so the suite is proven to catch violations before the first real adapter exists.

#### Scenario: Suite passes a conforming reference provider
- **WHEN** the suite runs against the conforming reference provider
- **THEN** it passes

#### Scenario: Suite fails a deliberately violating provider
- **WHEN** the suite runs against a reference provider rigged to violate the contract (leak context, omit provenance, or ignore cancellation)
- **THEN** the corresponding suite check fails
