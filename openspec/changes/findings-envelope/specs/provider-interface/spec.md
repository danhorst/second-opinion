# provider-interface

## ADDED Requirements

### Requirement: Result findings are raw reviewer text
The result's findings field SHALL hold the reviewer's output as the reviewer produced it.
A provider MUST NOT parse, restructure, summarize, or reformat that output, and the field's name SHALL say that it is raw so no consumer mistakes it for structured data.

#### Scenario: Adapter returns output unaltered
- **WHEN** a reviewer produces output through any adapter
- **THEN** the result's findings field holds that output byte-for-byte

#### Scenario: Structure is added outside the provider layer
- **WHEN** a consumer needs individual findings rather than text
- **THEN** it obtains them from the engine, not from the provider contract, which offers no structured findings type
