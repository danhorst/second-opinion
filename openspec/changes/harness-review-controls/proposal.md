## Why

Coding harnesses currently have to separate findings from stderr provenance, cannot inspect the assembled request without starting a paid review, and cannot bound reviewer latency. Diff assembly also inherits Git presentation and external diff settings and can mistake a base for a path or option.

## What Changes

- Add opt-in `--json` output containing raw findings and full provenance, without parsing or ranking findings.
- Add `--dry-run` to emit the exact engine request as JSON without constructing a provider.
- Add `--timeout DURATION` to bound the provider call, including fallback attempts.
- Support `--` for literal file arguments and reject missing flag values early.
- Make diff assembly cancellable and disable color, external diff and text conversion; delimit revisions from paths.
- Correct stale provider documentation and teach the canonical skill these controls and provenance limitations.
- Add local assembly benchmarks, eliminate an avoidable full-input copy, and run hermetic checks in pull-request CI.
- Record remaining completeness, isolation, and performance gaps in a repository review.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-cli`: JSON output, request preview, timeout, argument handling, and deterministic diff assembly.
- `agent-skill`: harness controls and honest interpretation of unknown model identities.

## Non-goals

MCP, Ollama, parsed findings, automatic triage, author identity enforcement, provider transport changes, and completion of the separate findings-envelope proposal. No new runtime dependencies.

## Impact

CLI, target assembly, canonical skill, README, regression tests, and specs. Default text output and existing provider contracts remain compatible. The request preview is engine input, not a claim to expose a provider's internal system instructions.
