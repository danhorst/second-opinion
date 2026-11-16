## ADDED Requirements

### Requirement: HTTP response budgets
HTTP providers SHALL honor configured output budgets for success and error response bodies and return the output-limit sentinel without partial success when exceeded. Success body limits SHALL be separate from diagnostic truncation limits.

#### Scenario: Reconciled execution
- **WHEN** a request uses the shared executor and configured output budgets
- **THEN** input material identity is preserved and excessive output fails explicitly
