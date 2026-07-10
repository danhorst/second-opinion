# antigravity-adapter

## Why

The codex adapter is implemented but its real-binary conformance is blocked on the codex account's usage limit; the conformance suite still has no fully-verified real consumer.
The antigravity CLI (`agy`) is installed, authenticated, and callable — an adapter for it unblocks full real-binary verification of the suite today, and adds a second working transport-sharing adapter that pressures the interface the way the design intended.

## What Changes

- New package `internal/provider/antigravity` implementing `provider.Provider` by shelling out to `agy --print`.
- Cold invocation: run from an empty temp directory (process working directory — `agy` has no `-C` flag), `--sandbox` terminal restrictions, no `--add-dir` workspace grants.
- Material embedded in the prompt argument as a delimited block — `agy --print` does not read stdin (verified empirically).
- Provenance: the forced model when one is set (`--model`), else the documented `antigravity-default` marker — `agy` has no structured output naming what ran.
- Integration tests: the adapter passes the shared conformance suite against the real `agy` binary behind the `integration` build tag — the suite's first complete real-binary verification.

## Capabilities

### New Capabilities
- `antigravity-provider`: the antigravity adapter's provider-specific requirements — cold invocation, prompt-embedded material, model forcing, did-not-run classification.

### Modified Capabilities
None — the adapter consumes `provider-interface` and `provider-conformance` without changing either contract.

## Non-goals

- The `claude`, `gemini`, and `ollama` adapters.
- Unblocking the codex adapter's conformance (separate: waits on quota or API-key auth).
- Model-name validation — `agy --model` values pass through verbatim; a bad name fails loudly at review time.
- Streaming, structured findings, or triage (unchanged open questions).

## Impact

- New package `internal/provider/antigravity`; no changes to `internal/provider` or `providertest`.
- `openspec/config.yaml` external dependencies and `AGENTS.md` gain the `antigravity` (`agy`) provider — a multi-model harness (Gemini, Claude, GPT-OSS), which makes truthful provenance matter more, not less.
- Requires `agy` on PATH and an authenticated antigravity session for `make test-integration`; the default `make test` run stays hermetic.
