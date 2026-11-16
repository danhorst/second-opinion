## 1. CLI and assembly

- [x] 1.1 Implement JSON results, request preview, timeout, and argument validation.
- [x] 1.2 Harden Git diff assembly and propagate cancellation.

## 2. Verification

- [x] 2.1 Add CLI behavior tests using the engine seam, without mocking provider binaries.
- [x] 2.2 Add real-Git regression tests for diff configuration, revisions, and cancellation.
- [x] 2.3 Run vet, tests, race checks, build, formatting, and OpenSpec validation.

## 3. Documentation

- [x] 3.1 Update README, canonical skill, changelog, and review findings with remaining limitations.
- [x] 3.2 Sync the delta specs to the current specs and record validation results.

## 4. Performance and automation

- [x] 4.1 Measure file assembly before/after removing the whitespace-check copy; keep reproducible benchmarks.
- [x] 4.2 Add pull-request CI using the existing repository action versions, without provider credentials.
