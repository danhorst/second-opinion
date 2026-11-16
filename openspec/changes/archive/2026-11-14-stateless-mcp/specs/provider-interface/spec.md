## ADDED Requirements

### Requirement: Optional execution output budgets
Provider construction SHALL support optional stdout/stderr capture budgets without adding non-content fields to the review request. When configured, every subprocess invocation including fallback SHALL enforce the budgets, cancel work on overflow, and return a distinguishable output-limit error without a completed or truncated result. Omitted limits SHALL preserve existing CLI behavior.

#### Scenario: Budgeted invocation overflows
- **WHEN** provider stdout or stderr exceeds its configured capture budget
- **THEN** execution is cancelled, readers are cleaned up, and an output-limit error is returned without findings

#### Scenario: Existing CLI caller
- **WHEN** a provider is constructed without execution output budgets
- **THEN** existing CLI output behavior is unchanged
