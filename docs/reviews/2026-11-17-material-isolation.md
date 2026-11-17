# Local material isolation — 2026-11-17

## Implemented boundary

`--require-isolation local-material-only` and MCP `require_isolation` require a
client-side boundary after caller assembly. A shared policy gate checks the actual
constructed adapter before invoking Review. Unknown/absent capabilities default to
`unverified`; unknown requirements fail closed. No fallback satisfies an unsupported
requirement. Results report the effective boundary separately from unchanged provenance.
Discovery uses static metadata, reads no credentials, and starts no review; a test
checks it against each constructed adapter's capability.

The OpenAI-compatible adapter supports this boundary: it sends only prompt/material
as review content, with no harness or tool loop. It disables HTTP redirects even on
injected clients without modifying the supplied client. Tool/function requests,
including finish-reason markers, mixed text/tool responses and later choices, fail
without findings. Paths and URLs in text are never dereferenced.

Codex, Claude, and Antigravity remain `unverified` for this boundary. Their existing
cold invocation obligations and evidence are unchanged.

## Verification

- `make test` with writable Go caches outside the outer sandbox passed: Python
  packaging tests, `go vet ./...`, and all Go tests. Local HTTP fixtures need loopback
  bind permission; the outer sandbox denied that permission on the initial run.
- Policy tests cover supported, absent, unknown and unsupported capabilities,
  unknown requirements, no invocation on rejection, default compatibility, and
  receipts overriding a provider-supplied isolation claim.
- Actual HTTP adapter fixtures check exact supplied content despite a planted
  ambient file and a referenced URL; no ambient content or follow-up request occurs.
  Redirect statuses 301/302/303/307/308 are rejected, including with a permissive
  injected client. Tool/function-call responses produce no completed result.
- CLI and SDK-backed MCP tests cover requirement parsing/validation, not-started
  rejection, discovery, preview without provider construction, default unverified
  results, and successful local HTTP reviews through the production adapter.
- MCP schema tests include the new input, boundary metadata, discovery and error.
- `go test -tags integration -run '^TestOpenAICompatibleConformance$'
  ./internal/provider/openai_compatible -v -timeout 180s` **skipped** because
  SECOND_OPINION_API_KEY and SECOND_OPINION_API_MODEL are not configured. This is
  not a passing endpoint certification or a new cold-check result.

- Strict OpenSpec validation passed for this change and all 14 current specs.
  Delta specs were synced to current specs. `git diff --check` passed.

## Limits

The endpoint (including a local server), HTTP transport, credentials, and configuration
are trusted dependencies. This client does not control endpoint-added context,
server-side tools, model obedience, or an endpoint's own filesystem permissions.
Compromised client code and complete OS isolation are outside the boundary.
Local fixtures verify client mechanics, not remote model conformance. Existing
positive-control-gated real-provider tests remain necessary for endpoint deployment
certification. A successful cold probe alone cannot establish filesystem denial.
