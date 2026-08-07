# provider-discovery

## Why

The tool ships no default provider on purpose, so every first run must answer "what can I pass to `--provider`?" — and today the only way to find out is to trigger an error.
That is backwards for a human and actively wrong for an agent: a discovery path that exits 2 conflates "here is the list you asked for" with "your invocation was invalid."

## What Changes

- New `--list-providers` flag: prints the registered provider names to stdout and exits 0, without requiring a target or a provider.
- The usage text names the registered providers on the `--provider` line, so the block already printed on every usage error answers the question in place.
- Fix `skills/second-opinion/SKILL.md`, which tells agents to "run without a provider to see what is registered" — that produces "nothing to review" unless a target is also passed and `SECOND_OPINION_PROVIDER` happens to be unset.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `review-cli`: provider discovery becomes a supported operation with its own exit-0 path, and the usage text gains the provider list.

## Non-goals

- A default provider. Neutrality is the reason discovery is needed; supplying a default would trade the problem for the bias the project context forbids.
- Probing which provider binaries are actually installed. `--list-providers` reports what is registered, not what would run — probing introduces machine-dependent output and an implicit preference in probe order.
- Reordering the target check ahead of the provider check in `run`. Bare invocation correctly reports the missing target; once usage names the providers, that path is no longer a dead end.
- Provider capability metadata (models, locality, auth). A name list is what selection needs today.

## Impact

- `cmd/second-opinion`: one flag in `parseArgs`, an early exit in `run`, and a usage string that composes the registry list rather than hardcoding it.
- `skills/second-opinion/SKILL.md`: one corrected line; the embedded copy must stay byte-identical, so the skill test covers it.
- `internal/review`: none. `Providers()` already returns what is needed.
