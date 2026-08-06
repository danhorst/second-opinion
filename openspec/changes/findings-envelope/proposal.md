# findings-envelope

## Why

The shape findings take at the engine boundary is the last open question blocking the MCP front-end.
The CLI can pass raw reviewer text through to stdout because a human reads it; an MCP tool result is consumed by a program, so provenance and individual findings need somewhere to live that is not prose.

## What Changes

- New package-level type in `internal/review`: an envelope carrying the verbatim reviewer text, a best-effort list of parsed findings, and a parse status.
- The baked prompt specifies a line-oriented convention for how each finding opens — a sentinel, severity, location — chosen so the raw text still reads as prose to a human.
- Parsing lives in `internal/review`, next to the prompt that requests the format. Adapters do not parse.
- **BREAKING** (internal): `provider.Result.Findings` is renamed to `provider.Result.Raw`. Once a parsed `Finding` type exists, a string named `Findings` is misleading. No external surface changes.
- Unparseable output is not failure: it yields the raw text, an empty finding list, and a status saying so. Exit codes are untouched.
- Parse status is an enum, not a boolean — a boolean conflates "the reviewer produced nothing" with "the parser recovered nothing".
- Each finding carries its byte span into the raw text, which is what makes it individually addressable and what makes coverage computable instead of guessed at.
- Each parsed finding carries the reviewer's own severity under a name that marks it as the reviewer's claim, not a fact.

## Capabilities

### New Capabilities
- `findings-envelope`: the engine-side envelope — the finding-header convention, best-effort parse, span addressing, parse-status semantics, and how provenance rides along.

### Modified Capabilities
- `provider-interface`: the result's findings field is raw reviewer text by contract, renamed to say so; adapters are forbidden from parsing or reformatting it.
- `review-prompt`: the baked prompt gains the finding-header convention, still transport-neutral; an overridden prompt is expected to yield unparseable output.
- `provider-conformance`: the suite asserts raw passthrough is byte-exact, which is now load-bearing rather than incidental.

## Non-goals

- The MCP front-end. This change settles the shape it consumes; it does not build it.
- Triage — clustering, deduping, and ranking stay with the calling agent. The open question about where triage lives stays open.
- A novelty field. The engine has no honest value to put in one, and an empty field invites the caller to fill it in, which is the author-grading-own-homework failure the triage question names.
- Same-model enforcement.
- Changing CLI output. Findings still go to stdout verbatim, provenance to stderr.
- Structured output negotiated per provider. The convention is prompt-requested and provider-independent.

## Impact

- `internal/provider`: one field rename, propagated to all four adapters and their tests.
- `internal/review`: new envelope type and parser; the baked prompt's text changes, so its hash changes and provenance from before this change will not match after it.
- `internal/provider/providertest`: raw passthrough becomes an asserted requirement.
- `cmd/second-opinion`: prints the envelope's raw text; observable behavior unchanged.
- `AGENTS.md`: the architecture line for `internal/review`, and the named-and-uncovered entry that design D7 commits to.
- `openspec/config.yaml`: the findings-shape open question is answered and comes out.
- The finding-header convention becomes a compatibility constraint the moment MCP consumes it.
