# MCP harness setup verification — 2026-11-16

## Implemented

Opt-in install/uninstall for Codex, Claude Code, and Antigravity CLI/IDE; no-review doctor;
and a bounded stdio connector to the private stateless HTTP service. HTTP setup stores
only the private token file path. No provider or model is inferred from the client harness.

The installer directly parses and merges configuration instead of invoking harness CLIs,
so previews, conflict handling, and backups have consistent behavior. TOML comments and
formatting are not preserved; unrelated setting values are preserved and the original
file is saved in a private adjacent backup. Concurrent edits are checked before publication,
without a cross-process lock. Custom Claude configuration directories require an explicit
file path rather than guessing their version-dependent layout.

## Evidence

- `make test`: Python service/release tests, Go vet, and all default Go tests passed.
- `go test -race -timeout 60s ./...`: passed.
- Registration tests cover all three formats, user/project paths, idempotence, dry-run,
  conflicts, JSON ambiguity and unsafe file rejection, unrelated TOML settings, exact
  backup contents/private permissions, longer valid timeouts, and HTTP token-free output.
- Connector tests exercise real local HTTP and the real MCP server SDK for both supported
  protocol versions; request IDs, malformed/oversized responses, redirects, active
  cancellation, EOF, saturation, duplicate IDs, and output failure are covered.
- Built-binary smoke checks used temporary configurations for all three adapters:
  preview, install, repeated install, doctor with real stdio discovery, uninstall,
  and repeated uninstall all passed.
- A built-binary HTTP smoke check used a temporary token and actual local server:
  install, doctor through the connector, and uninstall passed. No review was invoked.
- OpenSpec change validation, formatting, and diff whitespace checks passed.
- An independent implementation review found an overly strict exact-timeout check;
  it was fixed and a regression test now accepts longer review timeouts.

## Limits

These checks did not launch Codex, Claude Code, or Antigravity to certify configuration
acceptance, reload behavior, trust prompts, or scope/managed-policy precedence. No paid
provider request ran, and doctor does not certify provider credentials or review quality.
Live launchd/systemd lifecycle and GUI behavior remain unverified. The original provider
isolation and process cleanup evidence limits in AGENTS.md remain unchanged. Windows
private token handling is unsupported, as with mcp init; the HTTP service targets macOS/Linux.

## Configuration references

- https://developers.openai.com/codex/mcp/
- https://code.claude.com/docs/en/mcp
- https://antigravity.google/docs/mcp
