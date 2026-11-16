## ADDED Requirements

### Requirement: Idempotent MCP token initialization
`second-opinion mcp init` SHALL create a cryptographically random token at the service default path, optionally overridden by --token-file or SECOND_OPINION_MCP_TOKEN_FILE. Newly created token files SHALL have mode 0600 and newly created directories SHALL have mode 0700. Creation SHALL publish a complete token atomically without replacing an existing path. Existing valid private user-owned tokens SHALL be preserved; invalid files SHALL fail without mutation. The command SHALL NOT print tokens, start services, or invoke providers, and SHALL report actionable setup steps. It SHALL reject root and unsupported service platforms.

#### Scenario: First setup
- **WHEN** no token file exists
- **THEN** init creates a 64-character hexadecimal secret from 32 random bytes and reports its location without its value

#### Scenario: Repeated or concurrent setup
- **WHEN** a valid token already exists or another init caller publishes one first
- **THEN** init preserves that token and reports successful validation

#### Scenario: Invalid existing file
- **WHEN** the target is a symlink, is not private and user-owned, or contains malformed token data
- **THEN** init fails without changing it or exposing its content
