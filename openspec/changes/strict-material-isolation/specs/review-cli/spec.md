## ADDED Requirements

### Requirement: CLI isolation controls and receipts
The CLI SHALL accept `--require-isolation local-material-only`, reject unknown values as usage errors, and fail unsupported execution with exit 1 before invoking the reviewer. JSON results and stderr provenance reports SHALL identify the effective isolation boundary. Dry-run SHALL record a supplied requirement without checking or claiming support. `--list-providers --json` SHALL return registered names and a provider-isolation mapping without executing reviews; plain listing SHALL remain unchanged.

#### Scenario: Strict preview without provider
- **WHEN** dry-run is requested with local-material-only and no provider
- **THEN** preview succeeds with the requested requirement and no assertion of an effective boundary

#### Scenario: Unsupported CLI review
- **WHEN** local-material-only is required from a harness adapter
- **THEN** exit 1 and a diagnostic identify unsupported isolation without a review

## MODIFIED Requirements

### Requirement: Machine-readable completed reviews
With `--json`, the CLI SHALL emit one JSON object with `schema_version` 1, `kind` `review`, `raw` containing the provider's findings string unchanged, and `provenance` containing `provider`, `model`, and the full `prompt_hash`. It SHALL NOT parse, count, or rank findings. Default text output and stderr provenance SHALL remain available. Failures SHALL use the documented exit codes and stderr, without a success object. Help and plain discovery SHALL retain their text formats; `--list-providers --json` SHALL return structured discovery with isolation metadata. JSON output SHALL reject invalid UTF-8 with exit 1 rather than silently replace bytes.

#### Scenario: Findings and provenance travel together
- **WHEN** a review completes with `--json`
- **THEN** stdout decodes to a review object preserving raw findings and complete provenance, including empty findings

#### Scenario: Review fails
- **WHEN** a provider fails during a JSON review
- **THEN** stdout contains no success object and the CLI exits 1 with a diagnostic on stderr

#### Scenario: Output cannot be delivered
- **WHEN** writing a preview or completed result fails
- **THEN** the CLI exits 1 and reports an output failure without claiming that the reviewer did not run


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
