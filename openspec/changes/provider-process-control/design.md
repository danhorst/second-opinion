## Context

The adapters use CommandContext with buffered pipe readers; parent termination alone does not guarantee prompt return or descendant cleanup.

## Goals / Non-Goals

Control invocation-owned processes without changing material or authentication. Do not claim cleanup of daemonized children that escape their process group or full Windows tree termination.

## Decisions

A shared Run helper configures a new process group on Darwin/Linux/BSD, replaces cancellation with group kill, and sets a two-second WaitDelay for inherited output pipes. It cleans remaining group members after Run returns, including normal parent exit. Other platforms retain CommandContext parent cancellation and bounded pipe waits. OS argument-limit errors retain their error chain and gain a reduce-material diagnostic; no guessed platform-independent input ceiling is imposed.

Tests use the test executable as a real child process, not a fake provider. Readiness handshakes establish that cancellation occurs after process startup. Build checks cover Windows fallback.

## Cold-reviewer guarantee

Lifecycle handling neither reads additional content nor changes provider requests. No context is added.

## Risks / Trade-offs

Detached children may escape the group; this remains explicitly unsupported. WaitDelay is a drain bound, not a review timeout. Provider latency limits still come from caller contexts.
