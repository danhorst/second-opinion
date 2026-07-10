# antigravity-adapter — tasks

## 1. Invocation and composition

- [ ] 1.1 Implement `composeInstruction` (prompt + delimited material) and `buildInvocation` (argv, cwd isolation) per design D1/D2
- [ ] 1.2 Unit-test: sandbox flag, print mode, forced-model presence/absence, no `--add-dir`, material verbatim in the composed instruction

## 2. Execution, provenance, errors

- [ ] 2.1 Implement `Review`: temp-dir cwd lifecycle, `exec.CommandContext`, stdout as findings, provenance per D3
- [ ] 2.2 Implement error classification per D4 with signature constants; unit tests for each sentinel

## 3. Conformance

- [ ] 3.1 Add the integration-tagged conformance test plus a provenance test (forced and unforced)
- [ ] 3.2 Run `make test-integration` with `agy` on PATH; adjust to what the real binary shows and record any discovered signatures

## 4. Wrap-up

- [ ] 4.1 `make test` and `gofmt -l .` clean; default run stays hermetic
- [ ] 4.2 Add `antigravity` to `openspec/config.yaml` external dependencies and locality line; update `AGENTS.md` Architecture and milestones
