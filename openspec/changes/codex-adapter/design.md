# codex-adapter — design

## Context

First real `Provider`. The contract (`internal/provider`) and suite (`providertest`) are in place and validated only against the in-process loopback; this change is where their assumptions meet a real harness.
The predecessor script's cold-reviewer bug — running `codex exec` in the caller's repo — is the specific failure this adapter must make impossible, and the conformance suite already encodes that as canary checks.

## Goals / Non-Goals

**Goals:**
- `internal/provider/codex` passing the full conformance suite against the real binary (`integration` tag).
- Cold invocation by construction, with deterministic white-box tests on the argv.
- Truthful provenance from codex's own event stream.
- The auth-fallback behavior the predecessor script had, expressed as a testable function.

**Non-Goals:** see proposal — other adapters, CLI/front-ends, prompt authoring, selection policy.

## Decisions

### D1 — Invocation is built by a pure function

`buildInvocation(req, model, workdir) (argv []string, stdin string)` constructs the full command line; `Review` only executes it.
The cold flags are therefore unit-testable without running codex:

```
codex exec -C <workdir> -c project_doc_max_bytes=0
      --sandbox read-only --skip-git-repo-check --ephemeral
      --json [-m <model>] <prompt>
```

with material on stdin (codex appends piped stdin as a block after the instruction).
`workdir` is a fresh `os.MkdirTemp` per review, removed afterward — the empty-tempdir + doc-suppression pair is deliberately redundant: either alone closes the instruction-file leak, together they also close "cwd is the caller's repo".

### D2 — Provenance and findings come from the JSONL event stream

`--json` makes codex print events to stdout as JSONL.
The adapter scans events for the session-configuration event (source of the model that actually ran) and the final agent message (the findings).
Parsing is defensive — unknown event types are skipped — and the exact shapes are pinned by the integration test against the real binary rather than assumed from documentation.
Alternative considered: `-o FILE` for findings plus stderr-banner scraping for the model. Rejected: two channels, one of them unstructured; JSONL is one structured channel for both.
If the integration test shows JSONL omits the model, the fallback is recording the *requested* model plus a `(default)` marker — a documented degradation, not silent invention.

### D3 — Fallback is a predicate on stderr, retried at most once

The ChatGPT-auth rejection is detected by matching codex's stderr against the known signature (`"not supported when using Codex with a ChatGPT account"`), held in one constant.
Fallback fires only when a model was forced, retries exactly once with no forced model, and the result's provenance reports what actually ran (D2 gives us that for free).
Every other failure maps to did-not-run: `exec.ErrNotFound` → `ErrUnavailable`, auth-classified stderr → `ErrAuth`, anything else → a wrapped error with codex's stderr attached.
The predicate is a pure function unit-tested against captured stderr text — the rejection can't be reproduced on demand in integration (it depends on the active account type).

### D4 — Constructor shape

`codex.New(opts ...Option) *Provider` with `WithModel(string)` as the only v1 option (forced model; empty means codex's default).
Provider name in provenance is the constant `"codex"`.
Binary name is fixed; a `WithBinary` option can come later if testing demands it — not speculatively.

### D5 — Cancellation via exec.CommandContext

`exec.CommandContext` kills the codex process on context cancellation; a pre-flight `ctx.Err()` check keeps the pre-cancelled path cheap.
This satisfies the contract's cancellation requirement without goroutine bookkeeping.

## Cold-reviewer guarantee

Preserved by construction and proven twice:
- **Mechanism (D1):** every invocation runs from an empty temp directory with `project_doc_max_bytes=0` and a read-only ephemeral sandbox — the three leaks the predecessor script had (auto-loaded instruction files, caller-cwd inheritance, repo readable from the sandbox) are each individually closed.
- **White-box proof:** unit tests assert the argv from `buildInvocation` carries all cold flags, deterministically, on every run of `make test`.
- **Black-box proof:** the conformance suite's canary checks run against the real binary under the `integration` tag — the first time they meet an actual model harness.

## Risks / Trade-offs

- [JSONL event schema drift across codex versions] → Defensive parsing plus the integration test as the schema pin; a shape change fails loudly in `make test-integration`, not silently in production.
- [Fallback signature is a string match on stderr] → Same fragility the predecessor script accepted, now confined to one constant with a unit test; if OpenAI rewords the error, forced-model reviews fail loudly with `ErrAuth` rather than silently reviewing on the wrong model.
- [Integration tests cost real API calls] → They only run under the `integration` tag, invoked deliberately; the suite is one review per check, not a benchmark.
- [Canary checks are probabilistic against a real model] → Accepted in the suite's design (provider-interface D6); the white-box argv assertions are the deterministic layer for this adapter.

## Open Questions

None new. The finding-envelope and same-model-enforcement questions stay open at the project level; this adapter feeds them truthful provenance either way.
