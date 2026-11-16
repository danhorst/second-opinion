## ADDED Requirements

### Requirement: Material provenance across front-ends
MCP and CLI JSON success provenance SHALL include the full material_hash supplied by the provider. The shared executor SHALL verify it against the exact input material. The existing MCP request receipt SHALL retain its material hash and remain consistent with provenance.

#### Scenario: Reconciled execution
- **WHEN** a request uses the shared executor and configured output budgets
- **THEN** input material identity is preserved and excessive output fails explicitly
