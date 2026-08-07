# review-cli

## Purpose

The command-line front-end: target assembly, unbiased provider selection, output channels, exit-code contract.

## Requirements

### Requirement: Two target modes
The CLI SHALL review either a list of files (concatenated in argument order) or the caller's git diff (`--diff [BASE]`, BASE defaulting to HEAD).
The two modes are mutually exclusive, and invoking with neither is a usage error.

#### Scenario: File review
- **WHEN** the CLI is invoked with one or more paths
- **THEN** the material is the files' contents concatenated in argument order, each preceded by a header naming its path

#### Scenario: Diff review
- **WHEN** the CLI is invoked with `--diff BASE`
- **THEN** the material is `git diff BASE` from the caller's working directory

#### Scenario: Empty material refused
- **WHEN** target assembly produces empty material (empty diff, empty files)
- **THEN** the CLI refuses to run the review and exits nonzero with a message saying there is nothing to review

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

### Requirement: Findings on stdout, provenance on stderr
The CLI SHALL write the findings verbatim to stdout and a one-line provenance report (provider, model, prompt hash) to stderr.

#### Scenario: Piped output is clean findings
- **WHEN** the CLI's stdout is piped to another program
- **THEN** the pipe receives only the findings, while provenance remains visible on stderr

### Requirement: Exit codes distinguish outcomes
The CLI SHALL exit 0 when the review completed — findings, including none, are data.
It SHALL exit 1 when the reviewer did not run, with the classified cause on stderr, and 2 when the invocation was invalid.

#### Scenario: Clean review exits zero
- **WHEN** the reviewer runs and reports findings (or none)
- **THEN** the CLI exits 0

#### Scenario: Reviewer unavailable exits one
- **WHEN** the provider classifies a did-not-run (binary missing, auth, usage limit)
- **THEN** the CLI exits 1 with the classified cause on stderr

### Requirement: Model passthrough
`--model` SHALL pass through to the selected provider as its forced model, with the provider's own semantics (fallback, rejection) unchanged.

#### Scenario: Forced model reaches the provider
- **WHEN** the CLI is invoked with `--model X`
- **THEN** the provider is constructed with X forced, and provenance reflects what actually ran
