## ADDED Requirements

### Requirement: Optional isolation execution policy
Callers SHALL be able to require `local-material-only` separately from the content-only review request. A provider lacking support SHALL return a distinguishable isolation-unsupported error before Review execution, with no result or fallback. Unknown requirements SHALL fail closed. Successful results SHALL carry the effective local boundary separately from reviewer provenance.

#### Scenario: Required boundary unsupported
- **WHEN** a caller requires local-material-only from an unsupported provider
- **THEN** the reviewer is not invoked and the caller receives the isolation-unsupported error

#### Scenario: Requirement omitted
- **WHEN** no isolation requirement is supplied
- **THEN** existing provider selection remains available and the result identifies its effective boundary
