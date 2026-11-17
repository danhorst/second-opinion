## Context
See proposal.md. Existing transports enforce bounded requests and stateless reviews. Existing token initialization validates private user-owned files.

## Goals / Non-Goals
Provide predictable registration and no-cost discovery. Do not run harness sessions or change reviewer settings.

## Decisions
Use narrowly scoped configuration adapters for Codex TOML and Claude/Antigravity JSON, preserving unrelated contents and backing up before atomic replacement. Native CLI registration was considered, but does not offer uniform conflict handling, exact previews, or surgical rollback; direct configuration also avoids executing a harness during setup. Refuse unsupported syntax rather than damaging it. Use user/project paths with harness-specific roots and explicit config overrides when necessary.

HTTP registrations launch our stdio connector so GUI clients need no shell environment secret injection. The connector forwards bounded JSON RPC requests concurrently to the stateless JSON endpoint, maps cancellation to request contexts, and disallows redirects/proxies. It is specific to this server, not a universal MCP proxy.

Doctor uses the MCP SDK against the registered command and checks initialization and tool discovery only. This exercises the actual transport without invoking a provider. Client-side timeout allows server review timeout plus transport overhead.

The cold-reviewer guarantee is unchanged: no registration or bridge code reads review material from disk, adds context, chooses models/providers, or grants trust. Doctor never performs reviews.

## Risks / Trade-offs
- Harness configuration drift → adapter fixtures and explicit live-acceptance evidence gap.
- Concurrent configuration edits → check original contents before publication; no claim of cross-process transactional locking.
- Secret handling → reuse private-file validation, bounded reads, no raw response bodies in diagnostics.
- Homebrew upgrades → retain stable opt executable path instead of resolving symlinks to versioned Cellar paths.

## Migration Plan
Opt-in commands only; existing registrations continue working. Uninstall removes matching entry; backups enable recovery. No service startup or token rotation.
