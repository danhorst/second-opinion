## ADDED Requirements

### Requirement: Safe-mode invocation
Claude invocations SHALL disable customizations using safe mode, suppress skills, use strict MCP configuration with no configured servers, and disable session persistence, in addition to disabling tools and setting sources. Authentication SHALL remain available.

#### Scenario: Customizations and MCP servers are configured
- **WHEN** the adapter constructs an invocation
- **THEN** safe-mode, skill suppression, strict MCP configuration and no-session-persistence flags are present

## MODIFIED Requirements

### Requirement: Material is embedded in the prompt
The adapter SHALL compose a single instruction from the review prompt and material in a clearly delimited block via a pure function, and SHALL send it on stdin without a positional prompt argument, avoiding the OS command-line argument limit for material.

#### Scenario: Composed instruction is auditable
- **WHEN** the adapter executes a review request
- **THEN** stdin is a deterministic function of the request and contains the material verbatim
