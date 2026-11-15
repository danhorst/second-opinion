## ADDED Requirements

### Requirement: Stateless MCP calling discipline
The canonical skill SHALL teach callers to discover registered providers, explicitly select a reviewer, supply complete material by value, preserve raw findings and provenance, and distinguish a completed empty review from a failed or lost call. It SHALL explain that retries can repeat paid work and that result retrieval, stored context, and implicit workspace reads are unavailable. It SHALL NOT treat a parsed findings list as necessary for MCP or imply unknown model identities prove independence.

#### Scenario: Lost review response
- **WHEN** the harness loses a response after submitting a review
- **THEN** the skill directs it to treat completion as unknown and weigh duplicate work before issuing a fresh request
