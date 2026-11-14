# codex-adapter — tasks

## 1. Invocation

- [x] 1.1 Implement `buildInvocation` (argv + stdin) with the cold flags per design D1
- [x] 1.2 Unit-test the argv: empty-tempdir `-C`, `project_doc_max_bytes=0`, `--sandbox read-only`, `--skip-git-repo-check`, `--ephemeral`, `--json`, forced-model presence/absence

## 2. Execution and parsing

- [x] 2.1 Implement `Review`: temp-dir lifecycle, `exec.CommandContext`, stdin wiring, stdout/stderr capture
- [x] 2.2 Implement JSONL event parsing per D2: model from the session-configuration event, findings from the final agent message; defensive against unknown event types
- [x] 2.3 Unit-test the parser against captured JSONL fixtures

## 3. Fallback and error classification

- [x] 3.1 Implement the ChatGPT-auth rejection predicate as a pure function with the signature in one constant; unit-test against captured stderr
- [x] 3.2 Implement retry-once-on-default per D3, provenance reporting the substituted model
- [x] 3.3 Map failures to sentinels: `exec.ErrNotFound` → `ErrUnavailable`, auth-classified stderr → `ErrAuth`, others wrapped with stderr detail; unit tests for each

## 4. Conformance

- [x] 4.1 Add the integration-tagged conformance test: `providertest.Conform` against `codex.New()`
- [x] 4.2 Run `make test-integration` with `codex` on PATH; adjust suite or parser to what the real binary shows (expected — first real consumer), and pin the JSONL shapes in fixtures
      — RESOLVED 2026-08-07: full conformance suite passes against codex-cli 0.147.0-alpha.6.5 (contract, cancellation, positive-control, both cold canaries). The real `--json` stream (`thread.started`/`item.completed`/`turn.started`/`turn.completed`) never carries a model field, on forced or unforced runs alike — confirmed by direct capture, not just against the two synthetic fixtures `parseEvents` was originally tested with. `findModel` was chasing a field that does not exist in the live binary's output; removed, and `parseEvents` now returns findings only. The current real shape is pinned as a new fixture in `codex_test.go` (`TestParseEventsRealCapture`) alongside the two pre-existing shapes, which stay as schema-drift tolerance cases.
- [x] 4.3 Verify provenance from a real run names the actual model
      — RESOLVED 2026-08-07: since JSONL never names the model, "the model that actually ran" is only knowable when one was requested. Provenance now follows the same documented-degradation pattern as the antigravity adapter: forced runs report the requested model (verified — `TestCodexProvenanceForced`, `gpt-5.4-mini` echoed exactly), unforced runs report the `codex-default` marker rather than a guess (verified — `TestCodexProvenanceUnforced`). `codex-provider`'s spec requirement is updated to match.

## 5. Wrap-up

- [x] 5.1 `make test` and `gofmt -l .` clean; default run stays hermetic
- [x] 5.2 Update `AGENTS.md`: check off the codex adapter milestone, add `internal/provider/codex` to Architecture
