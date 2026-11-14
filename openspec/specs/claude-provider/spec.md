# claude-provider

## Purpose

The claude adapter's provider-specific requirements: cold invocation including harness-global-context suppression, prompt-embedded material, JSON provenance extraction, did-not-run classification.

## Requirements

### Requirement: Claude runs cold by construction
The claude adapter SHALL invoke `claude -p` with its process working directory set to an empty temporary directory, all tools disabled (`--tools ""`), and all setting sources suppressed (`--setting-sources ""`), so the reviewer can see neither the caller's repository nor any project or user-level instruction and memory files.

#### Scenario: Invocation is isolated from caller and user context
- **WHEN** the adapter runs a review while the caller's working directory contains instruction files and the user has global memory files
- **THEN** the claude process runs from an empty temporary directory with tools and setting sources disabled, and the review passes the conformance suite's cold checks

#### Scenario: Reviewer cannot use tools
- **WHEN** the adapter constructs an invocation
- **THEN** it disables the built-in tool set entirely, so file reads are impossible rather than merely unprompted

### Requirement: Material is embedded in the prompt
The adapter SHALL compose a single instruction from the review prompt and material in a clearly delimited block via a pure function, and SHALL send it on stdin without a positional prompt argument, avoiding the OS command-line argument limit for material.

#### Scenario: Composed instruction is auditable
- **WHEN** the adapter executes a review request
- **THEN** stdin is a deterministic function of the request and contains the material verbatim

### Requirement: Provenance from the JSON result
The adapter SHALL request `--output-format json` and extract the findings from the `result` field and the model that ran from `modelUsage`, choosing the dominant entry by output tokens (harness helper models contribute negligible output).
The provider name SHALL be `claude`.

#### Scenario: Model that ran is reported
- **WHEN** a review completes
- **THEN** provenance names the model ID that produced the findings, taken from the harness's own usage report rather than from the request

### Requirement: Did-not-run classification
The adapter SHALL map failures onto the contract's sentinels: missing binary yields `ErrUnavailable`, authentication or login failures yield `ErrAuth`, usage-limit exhaustion yields `ErrUnavailable`, and a result envelope with `is_error` true is a did-not-run with the envelope's message preserved.

#### Scenario: Error envelope
- **WHEN** claude exits successfully but the JSON envelope reports an error
- **THEN** the adapter returns an error carrying the envelope's message and no result

### Requirement: Conformance behind the integration tag
The adapter SHALL pass the shared conformance suite against the real `claude` binary behind the `integration` build tag, with white-box unit tests asserting the cold flags — including the global-context suppression required by `provider-conformance`.

#### Scenario: White-box cold flags
- **WHEN** the adapter constructs an invocation
- **THEN** a unit test can assert `--tools ""`, `--setting-sources ""`, print mode, JSON output, and forced-model presence/absence without executing claude

#### Scenario: Real-binary conformance
- **WHEN** `make test-integration` runs with `claude` on PATH and an authenticated session
- **THEN** the adapter passes the full conformance suite against the real binary

### Requirement: Validate result envelopes
The adapter SHALL accept only a result envelope with success subtype, explicit is_error=false, and a present string result as a completed review. It SHALL preserve the result text without whitespace trimming.

#### Scenario: Empty but valid review
- **WHEN** a valid successful envelope contains an empty result string
- **THEN** the adapter returns a completed empty review

#### Scenario: Invalid envelope
- **WHEN** output is null, an unrelated object, lacks result fields, or indicates failure
- **THEN** the adapter returns an error rather than an empty successful review

### Requirement: Safe-mode invocation
Claude invocations SHALL disable customizations using safe mode, suppress skills, use strict MCP configuration with no configured servers, and disable session persistence, in addition to disabling tools and setting sources. Authentication SHALL remain available.

#### Scenario: Customizations and MCP servers are configured
- **WHEN** the adapter constructs an invocation
- **THEN** safe-mode, skill suppression, strict MCP configuration and no-session-persistence flags are present
