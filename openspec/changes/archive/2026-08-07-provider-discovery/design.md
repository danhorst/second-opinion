# provider-discovery — design

## Context

`review.Providers()` has returned the sorted registry since the CLI change; nothing is missing at the engine level.
What is missing is a front-end path to it that does not go through an error.
Today the list appears in exactly two places, both `exit 2`: the no-provider-selected message and the unknown-provider message.

The CLI is consumed by agents as much as by humans, and the skill instructs agents to discover providers by running without one — advice that fails because `parseArgs` rejects a missing target before `run` reaches provider selection.

## Goals / Non-Goals

**Goals:**
- One invocation that answers "what providers exist" and exits 0.
- The usage block, already printed on every usage error, answers the question without a second invocation.
- The skill tells agents something that works.

**Non-Goals:** see proposal — no default provider, no installed-binary probing, no capability metadata, no reordering of the target and provider checks.

## Decisions

### D1 — Both a flag and the usage text, not one or the other

They serve different callers.
`--list-providers` gives an agent a deterministic, exit-0, parseable path — the thing an error message can never be, because exit 2 means the invocation was invalid and a successful query is not.
Naming providers in the usage text costs one line and fixes the human case at the moment of confusion, including bare invocation, without a second command.

Alternative considered: usage text only.
Rejected — it leaves agents parsing an error stream to learn configuration, which is the defect this change exists to remove.

Alternative considered: flag only.
Rejected — bare `second-opinion` would still print a usage block that mentions `--provider P` without ever saying what `P` can be.

### D2 — `--list-providers` short-circuits before every other check

The flag exits before target assembly and before provider selection, so it works with no target, no `--provider`, and no `SECOND_OPINION_PROVIDER`.
It is a query about the binary, not a review, and nothing about the caller's repo or environment should be able to make it fail.

### D3 — Output is one name per line

Bare names, newline-separated, no header and no decoration, so `second-opinion --list-providers | while read p` works.
The bracketed Go slice formatting in the current error messages (`[antigravity claude codex]`) is a debug artifact and is not the shape to commit to on stdout.

Error messages keep their existing format — they are prose for humans, and changing them is out of scope.

### D4 — Usage text composes the list from the registry

The `--provider` line renders `review.Providers()` rather than hardcoding names, so adding the `ollama` adapter updates the usage text by adding its registry line and nothing else.
This makes `usage` a function or a composed string rather than a constant.

### D5 — Registered, not installed

The list reports what the binary knows how to construct.
Whether `codex` is on PATH is a separate question, answered when a review actually runs, and answering it here would make output vary by machine and put an implicit vendor preference into whatever order the probe used.

## Cold-reviewer guarantee

Untouched.
No reviewer runs on this path, no material is assembled, and no provider is constructed — `--list-providers` reads a compile-time map and returns.

## Risks / Trade-offs

- [`usage` stops being a constant] → Small mechanical change; the tests that assert usage content move to asserting it contains each registered provider, which is the property worth pinning anyway.
- [One-name-per-line becomes a compatibility promise once agents parse it] → Deliberate, and the reason the format is specced rather than left to `fmt` defaults.

## Open Questions

None.
