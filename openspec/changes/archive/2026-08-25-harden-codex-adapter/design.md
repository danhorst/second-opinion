## Context

The Codex adapter already runs from a temporary directory and disables project-document loading, but the current command does not suppress user configuration or execution-policy rules.
Its read-only sandbox prevents writes; it does not provide the strict material-only filesystem boundary reserved for a later change.

The adapter also accepts any zero-exit JSONL stream as a successful review and falls back to guessed provenance when the event schema is not recognized.
That makes a Codex CLI upgrade capable of silently producing an empty or unverifiable review.

## Goals / Non-Goals

**Goals:**

- Prevent Codex user configuration, project documents, and execution-policy rules from supplying implicit review context.
- Keep the current guarantee precise: the adapter prevents implicit context, but does not claim to prevent all deliberate filesystem reads.
- Make successful Codex responses fail closed when required event data is absent or unrecognized.
- Preserve actual model provenance through forced-model fallback.
- Make the reviewed material auditable through a shared material hash.
- Exercise the behavior with deterministic parser/invocation tests and real-binary conformance tests.

**Non-Goals:**

- Strict filesystem isolation or a material-only execution boundary.
- MCP, same-model enforcement, structured finding envelopes, or triage.
- Provider selection policy or new providers.

## Decisions

### D1 — Suppress all known Codex ambient configuration

Add `--ignore-user-config` and `--ignore-rules` alongside the existing empty cwd, `project_doc_max_bytes=0`, read-only sandbox, skip-repo-check, and ephemeral flags.
These flags address distinct ambient inputs: user configuration and rules are separate from project documents and session persistence.

The adapter will not set a temporary `CODEX_HOME`, because that would also change authentication lookup and could make an otherwise authenticated Codex installation unusable.

### D2 — Treat event parsing as a protocol boundary

Change the parser result to record whether a recognized agent message and model were observed.
A zero-exit process with no recognized final message is a protocol error, not an empty review.
The current Codex JSONL stream does not expose the selected model for unforced runs, so those valid responses report the explicit `codex-default` provenance marker.
A forced review may retain the forced model only when the command was explicitly forced and the event stream otherwise contains a valid response.

Error extraction will walk nested JSON objects so top-level errors and nested turn-failure errors participate in classification.

### D3 — Keep fallback narrow and cancellation-safe

Retry once without `-m` only when a forced-model invocation fails with the known ChatGPT-account model rejection and the context is still active.
Usage limits remain unavailable; authentication failures remain authentication errors; unrelated failures are not retried.

The fallback predicate remains centralized and unit-tested against both top-level and nested captured events.

### D4 — Add material identity to shared provenance

Add a canonical `HashMaterial` helper and `MaterialHash` to `provider.Provenance`.
All existing adapters populate it from the exact `Request.Material` value, so the shared contract remains transport-neutral and every successful result is auditable.

### D5 — Test the installed Codex contract at two levels

Unit tests assert the complete invocation and parser/error behavior without mocking the provider boundary.
Integration tests behind the existing `integration` build tag run the shared conformance suite and provenance smoke test against the installed binary.
The tests document that real-binary verification is required for a Codex CLI upgrade.

## Risks / Trade-offs

- [Codex renames or removes isolation flags] → The integration test and versioned command capture fail before release; the adapter must be updated rather than silently dropping protection.
- [Codex does not expose the selected default model] → Report `codex-default` as a documented provenance degradation; forced-model runs retain the explicit model identity.
- [The read-only sandbox is mistaken for strict isolation] → Documentation and the spec explicitly call this “no implicit context”; strict material-only execution remains a separate capability.
- [Adding material hashes changes every adapter] → The helper is shared and the conformance suite checks the value, keeping the cross-provider change small and uniform.
- [A model emits no textual findings] → The parser tracks message presence separately from message content, so a valid empty message remains distinguishable from an unrecognized stream.

## Migration Plan

No data migration is required.

Implement the shared provenance field and adapter updates together, then run the hermetic suite and the opt-in integration suite against the installed Codex binary.
If a Codex version lacks the required flags or event fields, do not ship a compatibility fallback that reintroduces implicit context; update the adapter or retain the prior release.

## Open Questions

The later strict-isolation change must choose between an OS/container boundary and a provider that exposes a no-tools/API mode.
This change intentionally does not resolve that choice.
