## Why

Cancelling only a provider parent process can leave descendants running and output pipes open. Oversized argument-based material currently fails with an opaque OS error.

## What Changes

- Share subprocess lifecycle handling across the three adapters.
- Kill the invocation process group on cancellation and after completion on Unix; bound pipe-drain waits.
- Preserve context errors and explain argument-size failures without disclosing input.
- Test real subprocess trees independently of provider behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `provider-interface`: bounded subprocess cancellation and actionable argument-limit errors.

## Non-goals

MCP, new providers, streaming transport, escaped daemon cleanup, and Windows Job Objects.

## Impact

New internal process helper; small adapter wiring changes; no dependencies or mocked reviewer binaries.
