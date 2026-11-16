## Why

MCP reports version 1, which cannot identify the installed build during release validation.

## What Changes

Share granular application version reporting between CLI and MCP; stamp source archives at release.

## Capabilities

### Modified Capabilities
- `review-cli`: application build identity.

## Non-goals

Publishing a release, changing protocol/schema versions, or changing review behavior.

## Impact

A small buildinfo package, --version, MCP identity, and release-script stamping.
