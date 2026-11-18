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
The CLI SHALL provide `--list-providers`, which, unless combined with `--json`, writes the registered provider names to stdout, one per line with no header or decoration, and exits 0.
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
In default text mode, the CLI SHALL write the findings verbatim to stdout and a one-line provenance report (provider, model, prompt hash) to stderr.

#### Scenario: Piped output is clean findings
- **WHEN** the CLI's text-mode stdout is piped to another program
- **THEN** the pipe receives only the findings, while provenance remains visible on stderr

### Requirement: Exit codes distinguish outcomes
The CLI SHALL exit 0 when the review completed — findings, including none, are data.
It SHALL also exit 0 for a successful preview, help, or discovery operation. It SHALL exit 1 when the review fails or its output cannot be delivered, with the cause on stderr, and 2 when the invocation was invalid. A failed output write SHALL NOT be described as a reviewer that did not run.

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

### Requirement: Machine-readable completed reviews
With `--json`, the CLI SHALL emit one JSON object with `schema_version` 2, `kind` `review`, `raw` containing the provider's findings string unchanged, and `provenance` containing `provider`, `model`, and the full `prompt_hash`. It SHALL NOT parse, count, or rank findings. Default text output and stderr provenance SHALL remain available. Failures SHALL use the documented exit codes and stderr, without a success object. Help and plain discovery SHALL retain their text formats; `--list-providers --json` SHALL return structured discovery with isolation metadata. JSON output SHALL reject invalid UTF-8 with exit 1 rather than silently replace bytes.

#### Scenario: Findings and provenance travel together
- **WHEN** a review completes with `--json`
- **THEN** stdout decodes to a review object preserving raw findings and complete provenance, including empty findings

#### Scenario: Review fails
- **WHEN** a provider fails during a JSON review
- **THEN** stdout contains no success object and the CLI exits 1 with a diagnostic on stderr

#### Scenario: Output cannot be delivered
- **WHEN** writing a preview or completed result fails
- **THEN** the CLI exits 1 and reports an output failure without claiming that the reviewer did not run

### Requirement: Explicit request preview
With `--dry-run`, the CLI SHALL assemble the same prompt and material used by a review and emit JSON with `schema_version` 1 (or 2 when an isolation requirement is supplied), `kind` `request`, `prompt`, `material`, and `prompt_hash`. It SHALL NOT construct or invoke a provider and SHALL ignore provider/model selection. Successful preview SHALL exit 0; missing or empty targets SHALL retain their existing error behavior. Preview SHALL represent engine input rather than provider-internal instructions. A preview containing invalid UTF-8 SHALL fail with exit 2 instead of silently replacing bytes.

#### Scenario: Preview without credentials or provider
- **WHEN** a caller previews a file or diff, optionally with a prompt override
- **THEN** the exact assembled engine input is returned without starting a reviewer

### Requirement: Caller-controlled review deadline
The CLI SHALL accept `--timeout DURATION`, a strictly positive Go duration, and apply one deadline to the provider call including any retries. Omission SHALL preserve unlimited review duration. The timeout SHALL NOT include target assembly.

#### Scenario: Review exceeds the budget
- **WHEN** the provider call exceeds the requested timeout
- **THEN** its context is cancelled and the CLI exits 1 with no completed result

#### Scenario: Invalid duration
- **WHEN** timeout is empty, zero, negative, or invalid
- **THEN** the CLI exits 2 before invoking a reviewer

### Requirement: Unambiguous arguments and diff assembly
The CLI SHALL support `--` to end flag parsing and SHALL reject empty or flag-shaped required option values. Diff assembly SHALL reject option-shaped bases, separate revision arguments from paths, disable color, external diff and textconv, and honor caller cancellation. Tracked staged and unstaged changes against the base SHALL be included; untracked files SHALL NOT be implied to be included.

#### Scenario: Literal option-shaped filename
- **WHEN** a file named `--help` follows `--`
- **THEN** it is reviewed as a file instead of interpreted as a flag

#### Scenario: Missing value precedes a flag
- **WHEN** `--model --diff` is passed
- **THEN** it is rejected as a missing model value

#### Scenario: Git customization cannot replace the patch
- **WHEN** Git has color enabled or external diff or textconv configured
- **THEN** assembly returns an uncolored built-in patch without running those helpers

#### Scenario: Path mistaken for revision
- **WHEN** a diff base names only a file, not a revision
- **THEN** assembly fails instead of silently reviewing that path

### Requirement: Application build identity
The CLI SHALL expose `--version` and MCP SHALL report the same granular application version. Source archive releases SHALL embed the selected release version; Git builds SHALL include available revision and modified-state metadata. Application versions SHALL remain independent of protocol and output schema versions.

#### Scenario: Source archive installation
- **WHEN** a release source archive is built without Git metadata
- **THEN** the application reports its stamped release version

#### Scenario: Development checkout
- **WHEN** a binary contains Go VCS build metadata
- **THEN** reported application identity includes its revision and dirty state when modified

### Requirement: CLI isolation controls and receipts
The CLI SHALL accept `--require-isolation local-material-only`, reject unknown values as usage errors, and fail unsupported execution with exit 1 before invoking the reviewer. JSON results and stderr provenance reports SHALL identify the effective isolation boundary. Dry-run SHALL use schema version 2 when a requirement is supplied and retain version 1 otherwise. It SHALL record a supplied requirement without checking or claiming support. `--list-providers --json` SHALL use schema version 2 and return registered names and a provider-isolation mapping without executing reviews; plain listing SHALL remain unchanged.

#### Scenario: Strict preview without provider
- **WHEN** dry-run is requested with local-material-only and no provider
- **THEN** preview succeeds with the requested requirement and no assertion of an effective boundary

#### Scenario: Unsupported CLI review
- **WHEN** local-material-only is required from a harness adapter
- **THEN** exit 1 and a diagnostic identify unsupported isolation without a review
