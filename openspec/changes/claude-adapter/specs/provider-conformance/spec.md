# provider-conformance

## ADDED Requirements

### Requirement: Harness-global context is part of the cold guarantee
The cold-reviewer guarantee SHALL cover harness-global context — user-level instruction files, memory files, and settings the provider's tooling loads from outside the working directory.
Because global-context locations are harness-specific and the shared suite cannot plant canaries in a user's real global configuration, each adapter SHALL suppress its harness's global context and prove the suppression with deterministic white-box tests on the constructed invocation.

#### Scenario: Adapter suppresses global context
- **WHEN** an adapter's harness supports user-level instruction or memory files
- **THEN** the adapter's invocation disables loading them, and a unit test asserts the suppression without executing the harness

#### Scenario: Harness without global context
- **WHEN** an adapter's harness has no user-level context mechanism
- **THEN** the adapter documents that fact where its cold behavior is described, so the absence is a recorded decision rather than an unexamined gap
