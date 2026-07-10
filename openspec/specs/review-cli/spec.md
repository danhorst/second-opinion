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

#### Scenario: No provider specified
- **WHEN** the CLI runs without `--provider` and without the environment variable
- **THEN** it exits with a usage error naming every registered provider

#### Scenario: Unknown provider
- **WHEN** the CLI is given a provider name that is not registered
- **THEN** it exits with a usage error naming every registered provider

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
