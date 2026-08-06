# conformance-positive-control — design

## Context

The conformance suite's cold checks are all negative: the reviewer must not see an auto-loaded instruction file, must not read the caller's repository. Negative assertions are satisfied by a provider that does nothing at all.

That is not hypothetical. The antigravity adapter passed those checks on 2026-07-10 and passed them again today, while sending no material to the reviewer. The failure surfaced only because a human read the findings and noticed the reviewer was answering the word `--sandbox`.

## Goals / Non-Goals

**Goals:**
- A conformance check that fails when material does not reach the reviewer.
- Cold checks that mean what they claim, by being paired with proof the reviewer received something.
- The antigravity invocation fixed, with the test that keeps it fixed.

**Non-Goals:** see proposal — convention conformance, codex provenance, instruction composition.

## Decisions

### D1 — A canary in the material, asserted in the output

The suite includes a distinctive token in the material and requires it to appear in, or be demonstrably reflected by, the reviewer's output.

The token must be something a reviewer would echo when following instructions but would never produce spontaneously. The material instructs the reviewer to repeat it verbatim, and the prompt for this subtest is not the adversarial review prompt — this check is about transport, not about review quality.

Alternative considered: assert the output is merely non-empty.
Rejected — agy returned 200 non-empty, confident, entirely unrelated characters. Non-emptiness is exactly the assertion that already passed.

Alternative considered: assert output length or structure.
Rejected for the same reason, one step weaker.

### D2 — Cold checks are conditional on the positive control

The suite runs the positive control first and reports a distinct failure when it fails, rather than letting cold checks report success beneath it.

A cold check passing while the positive control fails is not a partial pass; it is a meaningless result. The suite says so rather than emitting a green line that a milestone will later cite.

### D3 — The antigravity fix is argument order, not a new flag

```
agy --sandbox [--model M] --print <instruction>
```

`--print` takes the prompt as its value, so it must sit immediately before the instruction. Verified against the real binary: this form returns the expected answer, and the current form returns a reply to `--sandbox`.

The spec gains a requirement that the instruction is placed where `agy --print` reads it. Pinning the general fact rather than the literal argv keeps the spec about behavior while still forbidding the regression.

### D4 — White-box assertions move from presence to semantics

The existing scenario asserts a unit test "can assert the sandbox flag, print mode, the composed instruction." The adapter satisfied all three while broken, because each was present somewhere in the argv.

The assertion becomes positional: the instruction must be the value `--print` consumes. That is checkable without executing agy, which is the point of a white-box test.

**This generalizes.** Any adapter whose harness takes the prompt as a flag value has the same failure mode, so the requirement is written about the adapter's own flag semantics rather than as an agy special case.

## Cold-reviewer guarantee

Strengthened, not weakened. The guarantee has always had two halves — the reviewer sees the material, and the reviewer sees nothing else — and only the second half was tested. The positive control adds the first.

The canary is explicitly assembled into the material by the suite, so nothing about the check introduces implicit context.

## Risks / Trade-offs

- [A reviewer may paraphrase rather than echo the canary] → The subtest's material instructs verbatim repetition and uses its own prompt, not the adversarial one. If a model still paraphrases, the check is nondeterministic and must be named as such rather than loosened into meaninglessness.
- [One more real review per adapter on integration runs] → Accepted. The alternative is the current state, where the suite's headline guarantee is untested.
- [The fix and the test land together, so the test's value is unproven] → Mitigated by demonstrating the order: run the new control against the unfixed adapter and watch it fail, then fix. A positive control that has never failed is not yet a control.

## Open Questions

- Whether `claude` and `codex` have latent invocation defects the control will expose. Unknown until it runs; that is the point of running it.
