## Why

Manual token creation is awkward and can overwrite an existing service credential.

## What Changes

Add `second-opinion mcp init` with optional --token-file and the service token-file environment override. Create a cryptographically random token privately and preserve valid existing credentials. Print safe next steps, never the secret.

## Capabilities

### Modified Capabilities
- `homebrew-service`: idempotent credential initialization.

## Non-goals

Token rotation, printing secrets, starting services, editing harness settings, or publishing a release.

## Impact

CLI startup command, setup documentation, Homebrew caveats, and filesystem behavior tests.
