# claude-adapter

## Why

With the claude adapter, the tool's neutrality premise becomes demonstrable end-to-end: Claude-authored work reviewed by a different Claude model, through the same seam as every other provider.
The claude CLI also surfaced a gap worth fixing while it's fresh: harness-global context (user-level `CLAUDE.md`/memory) is an implicit-context category the conformance spec doesn't name, and the suite's canaries cannot generically test it.

## What Changes

- New package `internal/provider/claude` implementing `provider.Provider` by shelling out to `claude -p`.
- Cold invocation: empty temp-directory cwd, `--tools ""` (the reviewer physically cannot read files), `--setting-sources ""` (suppresses user/project/local settings and user memory — verified empirically 2026-07-10).
- Material embedded in the prompt argument — `claude -p` with a prompt argument does not read stdin (verified empirically).
- Truthful provenance from `--output-format json`: findings from `result`, the model that ran from `modelUsage` (dominant entry by output tokens).
- **Spec update** (`provider-conformance` delta): a new requirement naming harness-global context as part of the cold guarantee, suppressed per-adapter and proven white-box, because global-context locations are harness-specific and the shared suite cannot plant canaries in a user's real global config.
- Integration tests: full conformance against the real `claude` binary behind the `integration` build tag.

## Capabilities

### New Capabilities
- `claude-provider`: the claude adapter's provider-specific requirements — cold invocation including global-context suppression, prompt-embedded material, JSON provenance extraction, did-not-run classification.

### Modified Capabilities
- `provider-conformance`: adds the harness-global-context requirement — the cold guarantee covers user-level instruction and memory files, suppressed by each adapter and proven white-box.

## Non-goals

- The `gemini` and `ollama` adapters.
- Unblocking codex conformance (unchanged: quota or API-key auth).
- Same-model enforcement — but note this adapter makes the question concrete: a Claude-authored diff reviewed via this provider needs the model-level comparison, and truthful `modelUsage` provenance is what enables it.
- Cost accounting — the JSON result carries `total_cost_usd`; recording it is a front-end concern.

## Impact

- New package `internal/provider/claude`; one ADDED requirement in the `provider-conformance` main spec at sync time; no suite code changes (the new requirement is adapter-proven by design).
- `openspec/config.yaml` already lists the `claude` provider; the `claude -p` assumption it recorded is now empirically confirmed, with flags pinned.
- Requires `claude` on PATH and an authenticated session for `make test-integration`; default `make test` stays hermetic.
