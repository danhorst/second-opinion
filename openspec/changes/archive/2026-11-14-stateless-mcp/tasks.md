## 1. Protocol and engine foundation

- [x] 1.1 Pin a stable official Go MCP SDK 1.x release; verify 2026-07-28 and 2025-11-25 transport behavior with executable compatibility tests before relying on API names from main-branch docs.
- [x] 1.2 Add a transport-neutral review executor with explicit provider/model selection and prompt resolution, preserving the existing CLI contract through regression tests.
- [x] 1.3 Add optional provider-construction capture limits and a distinct output-limit sentinel; keep provider.Request content-only and default CLI behavior unchanged.
- [x] 1.4 Implement bounded stdout/stderr writers that cancel subprocess execution on overflow, clean up readers, and never return partial findings. Apply limits to fallback attempts too.

## 2. Tool contract

- [x] 2.1 Define closed review/list_providers input schemas and versioned success/error output schemas; validate exact types, Unicode, nonblank values, byte limits, and provider names.
- [x] 2.2 Implement review with fresh per-call provider construction, raw/provenance preservation, resolved prompt hash and material receipt, and structured/text JSON equivalence.
- [x] 2.3 Implement registry-only discovery with effective limits and deterministic tool ordering; no availability/authentication probes.
- [x] 2.4 Implement fixed safe error mappings, conservative tool annotations, and explicit unknown-completion semantics; preserve existing provider fallback without adding gateway retries.

## 3. Admission and cancellation

- [x] 3.1 Implement two nonqueued review slots and bounded transport admission without blocking stdio cancellation notifications.
- [x] 3.2 Enforce the earliest client/server/deadline context, per-request isolation, slot cleanup on every path, and five-second shutdown drain.
- [x] 3.3 Enforce transport, decoded input, provider capture, raw output, and encoded response limits; reject excessive input before invocation and never silently truncate.

## 4. Transports and command

- [x] 4.1 Add `second-opinion mcp serve`, default stdio, with clean stdout, explicit startup exit behavior, and EOF/SIGTERM handling.
- [x] 4.2 Add opt-in loopback HTTP `/mcp` using the SDK stateless handler, no event store/session persistence, bounded network timeouts, and no persistent GET event stream.
- [x] 4.3 Require per-request local bearer authentication, exact loopback binding/Host validation, and Origin rejection; prevent gateway-token logging and child-environment inheritance.
- [x] 4.4 Verify both supported protocol versions over both transports; keep legacy negotiation metadata separate from review state and fail unsupported transport configurations explicitly.

## 5. Verification

- [x] 5.1 Test pure validation, exact provider requests, full provenance, empty success, safe error mappings, Unicode/byte boundaries, output schemas, and structured/text equivalence.
- [x] 5.2 Test A-then-B statelessness, fresh-process equivalence, repeated IDs on separate connections, and absence of repository/client-context lookups.
- [x] 5.3 Use real SDK clients over stdio and loopback HTTP for discovery/calls, stdout purity, authorization, framing, protocol metadata, no session affinity, disconnects, and saturation.
- [x] 5.4 Test concurrent cancellation, no queued work, capacity recovery, shutdown, slow transport limits, and provider overflow with actual subprocesses; run race and bounded-memory stress tests.
- [x] 5.5 Run integration-tagged provider positive controls/cold checks plus a real MCP review for each certified provider. Report unavailable credentials and inconclusive cold checks explicitly; do not substitute mocked provider binaries.

## 6. Documentation and completion

- [x] 6.1 Add stdio and private HTTP harness setup examples, token handling, limits, provenance caveats, and lost-response/duplicate-cost guidance to README and the embedded skill.
- [x] 6.2 Reconcile the findings-envelope proposal's claimed MCP prerequisite without implementing or silently discarding its independent parser design.
- [x] 6.3 Update AGENTS.md with architecture, evidenced MCP milestone, and any remaining untested paths; update project context and changelog only once implemented.
- [x] 6.4 Run vet/tests/build/format/OpenSpec validation, then sync delta specs and archive when implementation and required evidence are complete.
