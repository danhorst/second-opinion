## ADDED Requirements

### Requirement: Validate result envelopes
The adapter SHALL accept only a result envelope with success subtype, explicit is_error=false, and a present string result as a completed review. It SHALL preserve the result text without whitespace trimming.

#### Scenario: Empty but valid review
- **WHEN** a valid successful envelope contains an empty result string
- **THEN** the adapter returns a completed empty review

#### Scenario: Invalid envelope
- **WHEN** output is null, an unrelated object, lacks result fields, or indicates failure
- **THEN** the adapter returns an error rather than an empty successful review
