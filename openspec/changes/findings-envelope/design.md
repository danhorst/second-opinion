# findings-envelope — design

## Context

`provider.Result` has carried `Findings string` with a comment admitting the structure was undecided, and provenance travelling outside it precisely because the text had no place to hold it.
The CLI got away with that: it prints the string and a human reads it.
An MCP tool result has no stderr, so provenance needs a home, and its consumer is a program that wants findings it can address one at a time.

Two facts constrain every option.
No provider emits structured findings for free — `agy --print` has no structured output at all, and the JSONL from `codex` and the JSON envelope from `claude` are transport wrappers whose findings content is freeform prose.
And `review-prompt` guarantees that a caller's `--prompt-file` replaces the prompt entirely, so any format the baked prompt requests is absent by design on overridden runs.

## Goals / Non-Goals

**Goals:**
- One envelope the MCP front-end can consume without further shape decisions.
- Findings addressable individually, without making the address authoritative.
- The CLI's output unchanged, and its raw text still worth reading.

**Non-Goals:** see proposal — MCP itself, triage, novelty, same-model enforcement, per-provider structured-output negotiation.

## Decisions

### D1 — The envelope lives in `internal/review`; adapters do not parse

The provider layer returns raw text plus provenance, exactly as it does now.
`internal/review` parses that text into the envelope once, on the way out.

Alternative considered: put `[]Finding` on `provider.Result` and let each adapter fill it.
Rejected — four adapters means four parsers and four ways to diverge, and the conformance suite would then have to assert parse behavior per adapter, which is asserting model behavior wearing an adapter's clothes.
An earlier draft offered "the `ollama` adapter ships with zero parsing code" as the test of this split. That is circular — D1 forbids adapters from parsing, so no adapter can fail it.
The falsifiable version is a CI check that fails if any package under `internal/provider/` imports the parser.

The parser sits next to `BakedPrompt` because the prompt defines the convention and the parser reads it.
Splitting them across packages would let one drift from the other silently.

### D2 — A line-oriented convention, not JSON

Each finding opens with a sentinel-prefixed header line; everything until the next header is that finding's body.

```
>>[blocking] internal/review/target.go:41
The guard runs after the provider is constructed, so a zero-byte target still
pays for a subprocess. Move the check above NewProvider.
```

The severity vocabulary is closed (`blocking`, `major`, `minor`), matched case-sensitively; the `>>` sentinel is required; and the parser anchors at line start.
Sentinel and severity must be plain ASCII with no interleaved zero-width or control characters, so a line cannot match as a header while reading as prose to a human, or the reverse. Homoglyph and bidi handling beyond that is an accepted residual risk, named alongside the other adversarial-material risks rather than silently assumed away.

The prompt already asks the reviewer to state the finding, cite the location, and rate severity, but this is a genuine new burden and not merely a tightening: expressing severity as content is one thing, and emitting `>>` plus one of three exact lowercase words at line start is another.
The compliance-over-depth risk below is the place that cost is weighed; it should not be pre-discounted here.

The sentinel exists because anchoring on `[severity]` alone false-positives on our own material.
An earlier draft of this document carried a bare `[blocking] …` example line, which means the first review of this very change would have parsed the example as a finding.
Any document *about* the convention hits that, and documents about the convention are exactly what this tool reviews.

The sentinel narrows the problem; it does not close it. See D8.

Alternative considered: ask for a JSON array and unmarshal.
Rejected on three counts.
Raw text is the CLI's product, and JSON on stdout would be a regression for the human reader — "raw always survives" only pays if raw is worth surviving.
Truncated JSON is unparseable in its entirety, whereas a truncated line list loses only its last finding; reviewers hit output limits often enough that this asymmetry decides it.
And — stated as an assumption, not an observation, because it has not been tested — a prose-heavy adversarial prompt asking for strict JSON is expected to draw prose-with-JSON-in-it from at least one provider in the roster. The first two reasons carry this decision on their own; if this one turns out to matter, test it rather than cite it.

### D3 — Degradation is an ordinary outcome, not an error

Unparseable output yields `Raw` populated, `Findings` nil, `Status` unparseable, and exit codes untouched.

This is forced, not chosen.
`--prompt-file` is a specced feature that by construction produces output the convention never asked for.
If unparseable were an error, one specced feature would break another.
The precedent is already in the tree: the antigravity adapter ships documented provenance degradation rather than failing reviews it cannot fully attribute.

`Status` is an enum, not a boolean, because a boolean cannot carry the states that actually occur:

| Status | Meaning |
| --- | --- |
| `empty` | the reviewer ran and produced nothing to parse |
| `parsed` | at least one finding was recovered |
| `unparseable` | `Raw` has content and no finding was recovered |

A boolean conflates the first and third. A reviewer that genuinely finds nothing but writes one sentence saying so produces non-empty `Raw` with zero findings — indistinguishable, under a boolean, from output the parser failed on.

**`unparseable` deliberately covers two different events**, and this is a limit rather than an oversight.
A reviewer that clearly reports no findings without using the sentinel, and a reviewer whose output is genuinely garbled, both land here.
Separating them would require pattern-matching prose for "I found nothing" — a heuristic read of model output, which is the same move rejected for a stored `partial` status in this decision and for convention conformance in D7.
So the value is documented as a shared bucket: `unparseable` means the parser recovered nothing, and a consumer MUST NOT infer the reviewer's intent from it. `Raw` is where intent is legible, and `Raw` is always there.

**Partial-ness is derived, not stored.**
Partial parses are kept, but the envelope does not claim a `partial` status, because the parser cannot honestly compute one: it has no way to know how many findings the reviewer intended.
What it can report is coverage — the spans in D8 make the unclaimed fraction of `Raw` computable by any consumer that cares.
A stored `partial` flag would be a guess wearing a type.

`Findings` is best-effort and never a faithful decomposition of `Raw`.
`Raw` is the authoritative record and the type's documentation says so.

### D4 — Severity is named as a claim; novelty has no field

`ClaimedSeverity` rather than `Severity`, so the type carries the distrust the project context demands: the reviewer finds, it does not rank, and severity is its guess.
A bare `Severity` field would read as fact to every downstream consumer.

There is no novelty field, and adding one later requires a spec.
The engine cannot compute novelty, so the field could only ever be filled by the caller — and under MCP the caller is frequently the model that wrote the material, which is the author grading its own homework.
An absent field cannot be filled in by accident.

### D5 — `Result.Findings` becomes `Result.Raw`

Once a `Finding` type exists, a string named `Findings` actively misleads.
Mechanical rename across four adapters, the conformance suite, and the CLI.

### D6 — The baked prompt's hash changes

Adding the convention changes `BakedPrompt`, so `HashPrompt` returns a different value.
Nothing persists provenance across runs, so there is no migration — but reviews recorded either side of this change are not comparable by prompt identity, which is the hash working correctly.

### D7 — Testing, and what cannot be tested

The parser is pure, so it is unit-tested directly against adversarial inputs: empty output, prose with no headers, partially conforming output, truncation mid-finding, a body line that looks like a header, CRLF endings, an unknown severity word.

The conformance suite gains one requirement — raw passthrough is byte-exact — because adapters not mangling the text is now load-bearing rather than incidental.

That assertion alone is not sufficient, because it sits at the wrong layer.
The guarantee this change makes is that the *envelope* holds provider output byte-for-byte, and an adapter-level check cannot detect mutation in the provider-to-engine handoff or in envelope construction.
So the engine gains its own test: feed known bytes — CRLF, leading and trailing whitespace, a line that looks like a header — through a provider into an envelope, and assert `Envelope.Raw` is identical.

The suite does **not** gain a "provider emits parseable findings" requirement.
That asserts model behavior, not adapter behavior, and it is nondeterministic.
Whether real reviewers actually follow the convention cannot be covered by a deterministic test; per the project's testing rule that goes in `AGENTS.md` as named-and-uncovered rather than faked.

### D8 — A finding's address is its byte span into `Raw`

```go
type Finding struct {
    Span            [2]int // byte range into Raw, header line included
    Text            string // == Raw[Span[0]:Span[1]]
    Location        string // parsed from the header; a view, not the record
    ClaimedSeverity string // parsed from the header; a view, not the record
}
```

**The span starts at the sentinel, not at the first body line.**
Excluding the header would leave its bytes permanently unclaimed, so coverage could never reach 100% even on perfectly conforming output — and a consumer could not tell that built-in bias from genuinely missed content, which defeats the metric.
Including it means `Text` repeats what `Location` and `ClaimedSeverity` already expose. That duplication is correct: `Text` is the verbatim record and the parsed fields are conveniences derived from it, which is the same ordering as `Raw` outranking `Findings`.

The proposal promises findings become individually addressable. Severity, location, and body do not deliver that: location may be empty and is not unique, and array position is not stable across parser versions, prompt revisions, or a re-run.

A span is deterministic, unique within an envelope, and computed rather than invented.
It also makes "`Raw` is authoritative" mechanically true instead of a documentation claim — a finding is literally a view into `Raw`, so the list cannot drift from the record.
And it is what makes D3's coverage derivable.

The span is an **envelope-local reference, not a finding identity**.
Two reviews of the same material produce unrelated spans; nothing may treat a span as stable across envelopes.

Text outside every span is unclaimed, and a reviewer's lead-in summary before the first header is the common case.
Preamble is unclaimed on the same terms as any other unmatched text — no special category, because the parser has no way to tell a lead-in from a finding it failed to recognize.

This is also the containment for D2's residual ambiguity.
A forged header does not lose content: the bytes remain in `Raw`, the spans show exactly how the text was divided, and a consumer can see the split that was made.

## Cold-reviewer guarantee

Preserved, and largely untouched — this change works on the reviewer's output, not its input.
Parsing happens after the reviewer has returned; nothing the parser does can add context to what the reviewer saw.

The one input-side change is the convention text in the baked prompt.
It is explicitly assembled like the rest of the prompt, it is visible in the material the caller can audit, and its presence is detectable through the changed prompt hash in provenance.
No repo access, no auto-loaded file, no session history enters through it.

## Risks / Trade-offs

- [Asking for a format makes reviews shallower — models can optimize for compliance over depth] → The convention is one header line over prose the prompt already required; the body stays free. Watch the first real reviews against pre-change output, and back the convention out if depth drops. This is the risk most likely to actually bite.
- [An MCP caller counts `Findings` and reports "3 findings" when the reviewer wrote 5] → `Findings` is documented as best-effort, `Status` is on the envelope, and span coverage makes the unclaimed fraction of `Raw` visible; the MCP change must surface `Raw` as the primary content and the list as secondary.
- [A forged header reassigns the following finding's body] → An earlier draft of this design claimed a false positive merely "costs an extra list entry while `Raw` stays correct." That was wrong, and the review caught it: because a body runs until the next header, a forged header **steals the next finding's body**. The sentinel makes forgery rarer and D8's spans make the resulting split inspectable, but the damage is contained rather than prevented.
- [Material under review is attacker-controlled, and reviewers quote it] → Prompt-injected header lines can therefore reach the parser by design. No unfenced convention survives adversarial material; this is accepted, bounded by `Raw` staying authoritative, and specced with explicit scenarios rather than left implicit.
- [The convention becomes a compatibility constraint the moment MCP consumes it] → Accepted, and the reason it is being specced now rather than invented inside the MCP change.

## Open Questions

- Where triage lives stays open, deliberately. This change makes findings addressable; it does not say who is allowed to rank them.
- Whether the MCP tool result leads with `Raw` or with `Findings` is the MCP change's call. The recommendation here is `Raw` primary, for the counting risk above.
