# review-cli

## ADDED Requirements

### Requirement: Provider discovery
The CLI SHALL provide `--list-providers`, which writes the registered provider names to stdout, one per line with no header or decoration, and exits 0.
The flag SHALL short-circuit before target assembly and before provider selection, so it succeeds with no target, no `--provider`, and no `SECOND_OPINION_PROVIDER`.
The list SHALL report the providers the binary can construct, not the provider binaries installed on the machine.

#### Scenario: Discovery needs nothing else
- **WHEN** the CLI is invoked with only `--list-providers`, outside a git repository, with no environment variable set
- **THEN** it prints the registered provider names one per line to stdout and exits 0

#### Scenario: Output is machine-readable
- **WHEN** a caller pipes `--list-providers` output into a line-oriented reader
- **THEN** each line is exactly one provider name, suitable for passing back as `--provider`

#### Scenario: Registered rather than installed
- **WHEN** a registered provider's binary is absent from PATH
- **THEN** it still appears in the list, and its absence surfaces only when a review is actually attempted

## MODIFIED Requirements

### Requirement: Unbiased provider selection
The CLI SHALL select the provider from `--provider`, falling back to the `SECOND_OPINION_PROVIDER` environment variable.
With neither set, the CLI SHALL exit with a usage error listing the registered providers — it ships no baked default.
The usage text SHALL name the registered providers on its `--provider` line, composed from the registry rather than hardcoded, so adding an adapter updates the usage text without a separate edit.

#### Scenario: No provider specified
- **WHEN** the CLI runs without `--provider` and without the environment variable
- **THEN** it exits with a usage error naming every registered provider

#### Scenario: Unknown provider
- **WHEN** the CLI is given a provider name that is not registered
- **THEN** it exits with a usage error naming every registered provider

#### Scenario: Usage text answers the question in place
- **WHEN** the CLI prints its usage block for any reason, including a bare invocation with no target
- **THEN** the `--provider` line names every registered provider

#### Scenario: New adapter appears without a usage edit
- **WHEN** a provider is added to the registry
- **THEN** it appears in the usage text and in `--list-providers` with no change to either output's own source
