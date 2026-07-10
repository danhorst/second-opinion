# claude-adapter — design

## Context

Third adapter, same seam, one spec-level addition.
Empirical facts pinned by probe on 2026-07-10 (claude CLI 2.1.206): `-p` with a prompt argument does not read stdin; `--setting-sources ""` suppresses user memory (verified against a machine with a real global `CLAUDE.md`); `--output-format json` returns a single envelope with `result`, `is_error`, and per-model `modelUsage`; `--tools ""` disables the built-in tool set entirely.

## Goals / Non-Goals

**Goals:**
- `internal/provider/claude` passing full conformance against the real binary.
- The strongest cold posture of the three adapters: no tools, no settings, no cwd context.
- Truthful model-level provenance from `modelUsage` — the input the same-model-enforcement open question needs.
- The `provider-conformance` spec gains the harness-global-context requirement this adapter surfaced.

**Non-Goals:** see proposal.

## Decisions

### D1 — Invocation: maximal suppression

```
claude -p --output-format json --tools "" --setting-sources "" [--model <model>] <instruction>
```

with `cmd.Dir` set to a fresh empty temp directory.
`--tools ""` makes repo reads impossible rather than unprompted — a stronger property than codex's read-only sandbox or agy's `--sandbox`.
`--setting-sources ""` closes the harness-global channel (user/project/local settings and memory).
`--bare` was considered and rejected: its description includes skipping keychain reads, which risks the auth path, and the probe showed `--setting-sources ""` alone achieves the needed suppression.
`buildInvocation` stays pure; instruction composition reuses the delimited-block shape established by the antigravity adapter.

### D2 — Findings and provenance from the JSON envelope

One `json.Unmarshal` into a typed envelope: `result` → findings, `is_error`/`subtype` → error path, `modelUsage` → provenance.
`modelUsage` is a map keyed by model ID with per-model token counts; the harness runs helper models for housekeeping (the probe showed a haiku entry with 12 output tokens beside sonnet's 550), so the adapter picks the entry with the most output tokens as the model that ran.
Documented heuristic, deterministic, and honest: the dominant model is the one that wrote the findings.
Alternative considered: requested-model echo like agy. Rejected — the harness offers ground truth; use it.

### D3 — Error classification

- `exec.ErrNotFound` → `ErrUnavailable`.
- Exit 0 with `is_error: true`, or non-zero exit with a parseable envelope → error carrying the envelope's message.
- Signatures ("log in", "authentication", "invalid api key") → `ErrAuth`; "usage limit" → `ErrUnavailable` (mirrors codex).
- Signature constants unit-tested as pure predicates; extended from whatever integration surfaces.

### D4 — Constructor mirrors the siblings

`claude.New(opts ...Option) *Provider`, `WithModel(string)` (passed as `--model`; aliases like "opus" or full IDs both pass through), provider name constant `"claude"`.
Third instance of the adapter shape — after this change the pattern is confirmed enough that `gemini`/`ollama` need no design novelty.

## Cold-reviewer guarantee

Preserved and extended:
- **Mechanism (D1):** empty-cwd + `--setting-sources ""` + `--tools ""` close, respectively: working-directory instruction files, harness-global context, and repository access of any kind.
- **White-box proof:** unit tests assert every suppression flag — including the global-context suppression the new `provider-conformance` requirement mandates.
- **Black-box proof:** the full conformance suite against the real binary.
The spec delta exists because this harness demonstrated a leak channel the suite cannot generically canary: the suite's temp-dir checks can't touch a user's real `~/.claude/CLAUDE.md`, so global-context suppression is proven white-box per adapter. The probe verified the flag actually works against a real global memory file before we relied on it.

## Risks / Trade-offs

- [`modelUsage` dominant-entry heuristic could misattribute] → Only if a helper model out-generates the primary, which inverts the meaning of "helper"; the integration provenance test pins current behavior.
- [JSON envelope shape drift across claude versions] → Typed unmarshal fails loudly; the envelope fixture is captured from the probe and the integration test re-pins on every run.
- [ARG_MAX bounds material size] → Same accepted v1 bound as antigravity.
- [Per-review cost is nontrivial (~$0.04 for a trivial call on sonnet)] → Integration runs are deliberate and small; cost surfacing is a front-end concern (the envelope carries `total_cost_usd` when wanted).

## Open Questions

None new — but D2's truthful provenance makes the same-model-enforcement question actionable for the first time: a caller can now compare its own model ID against what reviewed it.
