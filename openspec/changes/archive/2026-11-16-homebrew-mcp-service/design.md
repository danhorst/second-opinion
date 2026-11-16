## Decisions

Install `second-opinion-mcp-service` in libexec. It execs the adjacent installed binary with `mcp serve --transport http`, defaulting to 127.0.0.1:8765. Refuse root. Read a 32–4096 character bearer token from ~/.config/second-opinion/mcp-token (override SECOND_OPINION_MCP_TOKEN_FILE); require a regular nonsymlink file owned by the current user with permissions 0400 or 0600. Accept a single trailing newline; reject whitespace and invalid bearer characters. Do not source shell configuration. Export only the gateway token and exec; provider subprocesses already strip it. Optional SECOND_OPINION_MCP_LISTEN is still validated by the server.

The formula installs the launcher and uses opt_libexec so upgrades do not pin an old keg. Run at login with no automatic crash restart loop for missing credentials, no root requirement, and a stable Homebrew/system PATH plus ~/.local/bin. Token persistence is deployment configuration, not review history. No token is embedded in the service definition or logs. Homebrew service manager owns lifecycle; exec preserves signal delivery to the server.

Keep a canonical formula template in packaging/homebrew; release automation renders only validated version/checksum placeholders. This ensures future tap updates carry service support.

## Cold-reviewer guarantee

The launcher reads only explicit deployment credential configuration, adds no review material, and does not read repositories or request client context. Existing adapters remain responsible for isolation.

## Validation

Exercise launcher refusal, token parsing, argv/environment delivery and exit propagation through a test executable at the gateway boundary. Validate rendered formula syntax and Homebrew service metadata without installing or loading launchd. Live launchd/harness behavior and credential availability outside an interactive shell remain unverified here, per the user's machine constraint.
