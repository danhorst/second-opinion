# codex-adapter

## Why

The provider contract and conformance suite exist but have never met a real reviewer — every assumption in them is untested against an actual model harness.
The codex adapter is the first real `Provider`: it replaces the reviewer half of `dotfiles/claude/bin/second-opinion.sh` and, critically, fixes that script's live cold-reviewer bug (it runs `codex exec` in the caller's repo, where codex auto-loads `AGENTS.md` and can read the tree under a read-only sandbox).

## What Changes

- New package `internal/provider/codex` implementing `provider.Provider` by shelling out to `codex exec`.
- Cold invocation: run from an empty temp directory (`-C`), suppress project-doc loading (`-c project_doc_max_bytes=0`), read-only ephemeral sandbox (`--sandbox read-only --skip-git-repo-check --ephemeral`).
- Provenance extraction: parse codex's JSONL event stream (`--json`) for the model that actually ran; findings from the final agent message.
- Auth-aware model fallback: a forced model that codex rejects under ChatGPT-account auth is retried once on the provider default, and provenance reports what actually ran.
- Integration tests: the adapter passes the shared conformance suite against the real `codex` binary behind the `integration` build tag.

## Capabilities

### New Capabilities
- `codex-provider`: the codex adapter's provider-specific requirements — cold invocation flags, JSONL provenance extraction, auth-aware model fallback, did-not-run classification.

### Modified Capabilities
None — the adapter consumes `provider-interface` and `provider-conformance` without changing either contract.

## Non-goals

- The `claude`, `gemini`, and `ollama` adapters.
- Retiring `dotfiles/claude/bin/second-opinion.sh` (that happens with the CLI front-end, which needs this adapter first).
- Prompt authoring — the adapter transports whatever prompt the engine hands it; the baked adversarial prompt migrates with the CLI change.
- Same-model author/reviewer enforcement (open question; provenance from this adapter feeds it).
- Model *selection* policy — the adapter accepts an optional forced model and a default; choosing between them is a front-end concern.

## Impact

- New package `internal/provider/codex`; no changes to `internal/provider` or `providertest`.
- First real exercise of the conformance suite — suite adjustments discovered here are expected and cheap while there is one adapter.
- Requires the `codex` CLI on PATH for `make test-integration`; the default `make test` run stays hermetic.
