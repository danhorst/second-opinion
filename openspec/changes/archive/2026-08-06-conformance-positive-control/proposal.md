# conformance-positive-control

## Why

The antigravity adapter has been shipping a broken invocation while passing every test, including full real-binary conformance.
`agy --print` takes the prompt as its flag *value*, but the adapter builds `agy --print --sandbox … <instruction>`, so `--print` consumes `--sandbox` as the prompt and the material is never sent.
A dogfood review returned agy's reply to the literal string `--sandbox`, and the tool reported exit 0.

Two test gaps let that through, and both are the same mistake in different places.
Every cold-guarantee check asserts the reviewer *cannot see* something, and a reviewer receiving no material passes all of them vacuously.
The adapter's white-box test asserts the flags are *present*, not that they are ordered so the reviewer actually receives the instruction.

A suite that cannot distinguish "cold" from "did not run" is not proving the guarantee it claims to prove.

## What Changes

- New conformance requirement: a positive control. Plant a canary in the material and assert the reviewer's output reflects it, so a provider that discards the material fails.
- The cold-guarantee checks are re-framed as conditional on the positive control passing — proving the reviewer cannot see extra context only means something once it is proven the reviewer saw the material.
- `antigravity-provider` gains a requirement that the composed instruction is placed where `agy --print` reads it, and its white-box test asserts argument semantics rather than flag presence.
- Fix the antigravity invocation: `--print` immediately precedes the instruction.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `provider-conformance`: adds the positive control and makes the cold checks meaningful only alongside it.
- `antigravity-provider`: pins where the instruction sits in the invocation, and strengthens the white-box assertion from presence to semantics.

## Non-goals

- Asserting that a reviewer follows the finding-header convention. That is model behavior, nondeterministic, and already out of scope for the suite.
- Codex's model provenance regression — a completed review reported `codex-default`, meaning no model appeared in the event stream. Under investigation; it may warrant its own change.
- Re-auditing the claude adapter's invocation beyond what the positive control covers.
- Changing how any adapter composes its instruction.

## Impact

- `internal/provider/providertest/conform.go`: one new subtest, plus a rigged mode on the reference provider that discards material.
- `internal/provider/antigravity/antigravity.go`: argument order in `buildInvocation`, and its white-box test.
- All three real adapters must pass the new control under the integration tag; the milestone claims in `AGENTS.md` are unverified until they do.
- The positive control costs one extra real review per adapter on integration runs.
