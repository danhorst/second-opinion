## ADDED Requirements

### Requirement: Validate completed transport output
The adapter SHALL return a result only for well-formed JSONL with a recognized agent message followed by successful terminal completion. Explicit empty message strings SHALL be valid. It SHALL reject malformed, oversized, incomplete, and failed streams and SHALL NOT search arbitrary nested objects for findings.

#### Scenario: Completed empty findings
- **WHEN** an empty agent message is followed by successful completion
- **THEN** the adapter returns empty findings and provenance

#### Scenario: Invalid or incomplete stream
- **WHEN** parsing fails, a turn fails, or successful completion or a recognized message is missing
- **THEN** the adapter returns an error and no findings result

#### Scenario: Nonfatal item warning
- **WHEN** a well-formed warning item precedes a completed review
- **THEN** the warning does not invalidate the successful review
