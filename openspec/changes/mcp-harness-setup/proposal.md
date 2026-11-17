## Why
MCP setup currently requires hand editing harness configuration and moving service credentials. Provide reproducible registration and diagnostics without exposing tokens or selecting a reviewer implicitly.

## What Changes
- Add install/uninstall with dry-run and JSON output for Codex, Claude Code, and Antigravity.
- Add a bounded stdio-to-local-HTTP connector reading the protected token file.
- Add doctor to check registration and real MCP discovery without paid reviews.

## Capabilities
### New Capabilities
- `mcp-harness-setup`: safe registration, connector, and diagnostics.
### Modified Capabilities
None.

## Impact
CLI routing, a harness configuration package, MCP client transport, documentation and tests.

## Non-goals
No auto-trust, reviewer selection, service startup, paid diagnostic reviews, remote endpoints, or Cursor/VS Code support in this first slice. Live harness acceptance remains external validation.
