## Why

The shared cold check currently tests only relative file paths, and installed harnesses expose new context-suppression controls. Claude material need not occupy one OS command-line argument.

## What Changes

- Add a positive-control-gated absolute-path canary and nil-result checks to shared conformance.
- Disable Codex user configuration, rules, host skill discovery, memory, connectors, plugins, and local tool surfaces using controls observed in codex-cli 0.167.0-alpha.4.1.
- Add Claude safe mode, strict empty MCP configuration, disabled skills and session persistence; send composed material via stdin with no positional prompt.
- Install the canonical Codex skill in the documented native user-skill directory, preserving existing legacy prompt files.
- Attempt bounded real-binary conformance and report limitations explicitly.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `provider-conformance`: absolute-path cold check and robust invalid-result diagnostics.
- `codex-provider`: explicit suppression of user configuration and extra tools/context.
- `claude-provider`: safe mode and stdin transport.
- `agent-skill`: native Codex skill location and non-destructive migration.

## Non-goals

No fabricated isolation guarantee for Antigravity, no modification of the user's real global configuration, no provider-boundary mocks, no new providers or MCP front-end.

## Impact

Provider flags and tests, shared conformance, documentation. Requires harness versions supporting the selected flags; unsupported flags fail loudly rather than falling back to a warmer invocation.
