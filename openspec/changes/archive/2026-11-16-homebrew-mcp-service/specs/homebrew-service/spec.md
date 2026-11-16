## ADDED Requirements

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
