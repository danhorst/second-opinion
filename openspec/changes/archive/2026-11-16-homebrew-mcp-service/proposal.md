## Why

A shared local HTTP MCP gateway needs reliable per-user startup through Homebrew services, with credentials available outside interactive shell profiles.

## What Changes

Add an installed foreground launcher, a canonical formula template with a service block, release rendering, and setup instructions.

## Capabilities

### New Capabilities
- `homebrew-service`: private token-file startup and Homebrew service packaging.

## Non-goals

Publishing a new release, starting a service on this machine, creating credentials automatically, public hosting, or changing review semantics.

## Impact

Packaging scripts and release automation; no provider or MCP protocol changes. Current installed v0.2.0 remains unchanged until a release containing this packaging is published.
