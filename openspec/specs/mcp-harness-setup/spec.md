## Purpose
Make local MCP registration reproducible across supported harnesses while keeping credentials private and review selection explicit.

## Requirements

### Requirement: Safe harness registration
The CLI SHALL provide install and uninstall for codex, claude, and antigravity, with explicit harness, user scope by default, optional project scope, stdio by default, HTTP via the connector, dry-run and JSON output. Installation SHALL use an absolute executable path, accommodate 300-second reviews where supported, preserve unrelated settings, refuse conflicting registrations unless replacement is explicit, and be idempotent. Uninstall SHALL remove only a matching Second Opinion entry. Mutations SHALL be atomic with private backups for existing files and SHALL refuse unsafe file types and malformed configuration. Project configuration SHALL contain no credentials. Configuration SHALL NOT select a reviewer or grant trust.

#### Scenario: Repeat installation
- **WHEN** the desired registration already exists
- **THEN** installation succeeds without changing unrelated configuration or creating another registration

#### Scenario: Preview and conflicts
- **WHEN** installation is previewed or encounters a conflicting entry without explicit replacement
- **THEN** it makes no filesystem changes and reports the proposed change or conflict

#### Scenario: Removal
- **WHEN** uninstall finds a matching registration
- **THEN** it removes that entry only and leaves the service and token intact

### Requirement: Private local service connector
The CLI SHALL provide mcp connect as a bounded stdio bridge to the existing stateless JSON HTTP endpoint. It SHALL read a validated private token file, accept only literal loopback HTTP endpoints at /mcp, disable proxies and redirects, retain no history, and keep tokens and material out of diagnostics. EOF, cancellation and shutdown SHALL cancel outstanding HTTP work; request and response limits and bounded concurrency SHALL apply.

#### Scenario: Service discovery through connector
- **WHEN** a client initializes and lists tools through the connector
- **THEN** it receives service responses with original request identifiers and the token never appears in stdout

#### Scenario: Unsafe destinations
- **WHEN** an endpoint is remote, redirects, or the credential file is unsafe
- **THEN** connection fails without forwarding credentials to another destination

### Requirement: Non-review diagnostics
The CLI SHALL provide doctor with explicit harness and JSON output. It SHALL validate registration and executable availability, initialize MCP and discover review and list_providers, and report configuration validity separately from connectivity. It SHALL NOT invoke review or claim live harness acceptance or provider authentication has been verified. Failures SHALL return nonzero with actionable redacted diagnostics.

#### Scenario: Healthy installation
- **WHEN** the configured server initializes and lists required tools
- **THEN** doctor reports successful protocol discovery and explicitly identifies real review verification as not performed

#### Scenario: Failed connection
- **WHEN** configuration is valid but service authentication or startup fails
- **THEN** doctor distinguishes that connection failure from configuration success without revealing credentials
