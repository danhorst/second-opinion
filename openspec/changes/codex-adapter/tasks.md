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
- [ ] 4.2 Run `make test-integration` with `codex` on PATH; adjust suite or parser to what the real binary shows (expected — first real consumer), and pin the JSONL shapes in fixtures
      — BLOCKED 2026-07-09: contract and cancellation passed against the real binary; the remaining checks hit the codex account's usage limit (resets Jul 19). Error-event shapes were captured and pinned; the cold canaries and provenance need a fresh quota.
- [ ] 4.3 Verify provenance from a real run names the actual model

## 5. Wrap-up

- [x] 5.1 `make test` and `gofmt -l .` clean; default run stays hermetic
- [ ] 5.2 Update `AGENTS.md`: check off the codex adapter milestone, add `internal/provider/codex` to Architecture
