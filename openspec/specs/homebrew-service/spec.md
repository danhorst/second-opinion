# homebrew-service

## Purpose

Package the private MCP HTTP gateway as an optional per-user Homebrew service.

## Requirements


### Requirement: Private foreground service launcher
The launcher SHALL run the installed MCP HTTP server as the current non-root user, default to literal loopback, read a bounded bearer token from a private user-owned regular nonsymlink file, and export it without placing it in argv or logs. It SHALL reject invalid configuration before starting the server, SHALL NOT source shell profiles, and SHALL exec the server so signals and exit codes propagate.

#### Scenario: Unattended service startup
- **WHEN** the service starts with a valid private token file
- **THEN** the server receives the token through its environment and complete HTTP startup arguments

#### Scenario: Missing or insecure token
- **WHEN** the token file is absent, insecure, oversized, or malformed
- **THEN** startup fails with a fixed diagnostic that does not expose the token

### Requirement: Homebrew service packaging
The formula SHALL install the launcher, expose a per-user service using stable opt paths, provide an explicit executable search path, and keep secrets out of its generated service definition. Release automation SHALL preserve this service definition when rendering the tap formula.

#### Scenario: Formula upgrade
- **WHEN** Homebrew upgrades to a new release
- **THEN** the service command resolves the active installation and the token file remains outside the keg

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
