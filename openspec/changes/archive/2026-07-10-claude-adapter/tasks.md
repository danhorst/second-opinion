# claude-adapter — tasks

## 1. Invocation and composition

- [x] 1.1 Implement `composeInstruction` and `buildInvocation` per design D1
- [x] 1.2 Unit-test the cold flags: `--tools ""`, `--setting-sources ""`, print + JSON modes, forced-model presence/absence, material verbatim and delimited

## 2. Envelope, provenance, errors

- [x] 2.1 Implement the typed JSON envelope and `Review`: findings from `result`, provenance from dominant `modelUsage` entry per D2
- [x] 2.2 Unit-test envelope parsing against the captured probe fixture, including the helper-model heuristic
- [x] 2.3 Implement error classification per D3 with signature constants; unit tests for each sentinel and the `is_error` envelope path

## 3. Conformance

- [x] 3.1 Add integration-tagged conformance and provenance tests
- [x] 3.2 Run `make test-integration`; adjust to what the real binary shows

## 4. Wrap-up

- [x] 4.1 `make test` and `gofmt -l .` clean; default run stays hermetic
- [x] 4.2 Update `AGENTS.md` (Architecture entry, milestone) and confirm `openspec/config.yaml`'s claude line matches the pinned flags
