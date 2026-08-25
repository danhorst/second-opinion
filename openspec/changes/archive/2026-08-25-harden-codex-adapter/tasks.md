## 1. Shared provenance

- [x] 1.1 Add `HashMaterial` and `MaterialHash` to the provider contract.
- [x] 1.2 Update the Claude, Antigravity, and loopback providers to populate material provenance.
- [x] 1.3 Extend shared conformance tests to require the material hash.

## 2. Codex invocation hardening

- [x] 2.1 Add `--ignore-user-config` and `--ignore-rules` to the Codex invocation.
- [x] 2.2 Extend invocation tests to assert every ambient-context suppression flag.
- [x] 2.3 Update Codex comments and documentation to distinguish no implicit context from strict filesystem isolation.

## 3. Codex protocol and failure handling

- [x] 3.1 Make JSONL parsing report whether a recognized agent message and model were observed.
- [x] 3.2 Return a protocol error for incomplete successful streams while preserving valid empty findings.
- [x] 3.3 Extract nested error-event messages and expand parser fixtures for nested failures.
- [x] 3.4 Make model fallback cancellation-safe and verify it retries only the known auth rejection.
- [x] 3.5 Add tests for nested usage limits, authentication failures, malformed success output, and cancelled fallback.

## 4. Real-binary verification

- [x] 4.1 Run the shared Codex conformance suite against the installed authenticated binary.
- [x] 4.2 Pin the observed success event shapes and model provenance behavior in integration coverage, including the `codex-default` degradation.
- [x] 4.3 Record any supported Codex version or event-schema constraints discovered during verification.

## 5. Documentation and validation

- [x] 5.1 Update README, OpenSpec context, and AGENTS milestone notes to remove unsupported Codex claims.
- [x] 5.2 Run `gofmt`, `go vet ./...`, and the hermetic test suite with repository-local Go and Git caches where needed.
- [x] 5.3 Run strict OpenSpec validation and confirm all tasks are complete before implementation handoff.
