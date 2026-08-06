## 1. Prove the control catches the bug

- [x] 1.1 Add the positive control to `providertest.Conform`: material carrying a canary token, its own non-adversarial prompt instructing verbatim repetition, and an assertion that the reviewer's output contains the token.
- [x] 1.2 Add a `DiscardMaterial` mode to the loopback reference provider that ignores the request's material and returns unrelated non-empty output.
- [x] 1.3 Assert the positive control fails against that rigged provider — non-emptiness must not satisfy it.
- [x] 1.4 **Before fixing the adapter**, run the control against the real `agy` binary with the current invocation and record that it fails. A control that has never failed is not yet a control.

## 2. Fix the antigravity invocation

- [ ] 2.1 Reorder `buildInvocation` in `internal/provider/antigravity/antigravity.go` to `agy --sandbox [--model M] --print <instruction>`, so the instruction is the value `--print` consumes.
- [ ] 2.2 Rewrite the white-box unit test to assert position — the argument after `--print` is the composed instruction — rather than flag presence.
- [ ] 2.3 Add a white-box case with a forced model, asserting the model flag precedes `--print` and the instruction still immediately follows it.
- [ ] 2.4 Re-run the positive control against the real binary and confirm it now passes.

## 3. Make the cold checks honest

- [x] 3.1 Run the positive control before the cold checks in `Conform`.
- [x] 3.2 When the positive control fails, report the cold checks as inconclusive rather than passing.
- [ ] 3.3 Assert this reporting behavior against the `DiscardMaterial` provider: cold checks must not emit a pass beneath a failed control.

## 4. Re-verify the other adapters

- [ ] 4.1 Run the positive control against the real `codex` binary under the integration tag.
- [ ] 4.2 Run the positive control against the real `claude` binary under the integration tag.
- [ ] 4.3 Record the results; any adapter that fails gets its own fix, scoped separately from this change.

## 5. Documentation

- [ ] 5.1 Correct the `AGENTS.md` milestone claiming antigravity passed full real-binary conformance on 2026-07-10 — it passed a suite that could not detect this failure.
- [ ] 5.2 Note in `AGENTS.md` that a passing cold result is only meaningful alongside a passing positive control.

## 6. Verify

- [ ] 6.1 `make test` passes and `gofmt` is clean.
- [ ] 6.2 `make test-integration` passes for all three adapters, positive control included.
- [ ] 6.3 Re-run the dogfood review through antigravity and confirm the findings are about the material, not about a flag name.
