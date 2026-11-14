## Context

Malformed transport is currently indistinguishable from a clean empty review. Existing captured Codex fixtures show completed item events and turn.completed; Claude fixtures show type=result, subtype=success and a result string.

## Goals / Non-Goals

Validate transport success independently of findings content. Do not parse or rank findings or change model provenance.

## Decisions

Codex parsing returns text plus an error. Only recognized top-level event locations can supply findings: item.completed/item.agent_message and the legacy msg.agent_message envelope. A recognized terminal turn.completed (or legacy msg.task_complete) is required after a message. Empty string messages are valid. Reject malformed JSON, oversized scanner tokens, error/turn.failed events, missing result, and incomplete subsequent turns. Unknown well-formed events and nonfatal item warnings remain compatible. Error diagnostics never quote arbitrary material.

Claude validates the result discriminator, success subtype, explicit is_error=false, and a present string result. Error envelopes remain available for error classification. Raw result whitespace survives unchanged.

## Cold-reviewer guarantee

These changes inspect output only; no extra context reaches the reviewer.

## Risks / Trade-offs

New event protocol variants fail loudly until deliberately supported. Legacy support is limited to known envelope shapes. Empty final messages remain valid, but missing final messages are not treated as empty reviews.
