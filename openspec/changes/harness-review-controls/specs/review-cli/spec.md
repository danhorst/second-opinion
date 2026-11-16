## ADDED Requirements

### Requirement: Machine-readable completed reviews
With `--json`, the CLI SHALL emit one JSON object with `schema_version` 1, `kind` `review`, `raw` containing the provider's findings string unchanged, and `provenance` containing `provider`, `model`, and the full `prompt_hash`. It SHALL NOT parse, count, or rank findings. Default text output and stderr provenance SHALL remain available. Failures SHALL use the documented exit codes and stderr, without a success object. Help and discovery SHALL retain their text formats. JSON output SHALL reject invalid UTF-8 with exit 1 rather than silently replace bytes.

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
With `--dry-run`, the CLI SHALL assemble the same prompt and material used by a review and emit JSON with `schema_version` 1, `kind` `request`, `prompt`, `material`, and `prompt_hash`. It SHALL NOT construct or invoke a provider and SHALL ignore provider/model selection. Successful preview SHALL exit 0; missing or empty targets SHALL retain their existing error behavior. Preview SHALL represent engine input rather than provider-internal instructions. A preview containing invalid UTF-8 SHALL fail with exit 2 instead of silently replacing bytes.

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

## MODIFIED Requirements

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
