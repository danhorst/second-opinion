## 1. Provider contract

- [ ] 1.1 Rename `provider.Result.Findings` to `provider.Result.Raw` in `internal/provider/provider.go`; replace the "structure is deliberately undecided" doc comment with one stating the field is raw reviewer text that providers must not parse or reformat.
- [ ] 1.2 Propagate the rename through all four adapters (`codex`, `claude`, `antigravity`, `providertest.Loopback`) and their tests.
- [ ] 1.3 Audit each adapter's output path for trimming, re-wrapping, or injected headers; remove any reformatting that is not forced by the transport, and note in the adapter's doc comment anything the transport does force.

## 2. Conformance

- [ ] 2.1 Add a byte-exact raw passthrough check to `providertest.Conform`, using reviewer output with multiple lines, leading and trailing whitespace, and text resembling the finding-header convention.
- [ ] 2.2 Extend the deliberately-violating reference provider with a reformatting mode, and assert the passthrough check fails against it.
- [ ] 2.3 Confirm no conformance check asserts that a provider's output is parseable — that requirement is explicitly out of scope for the suite.

## 3. Prompt

- [ ] 3.1 Add the finding-header convention to `review.BakedPrompt`: header line carrying `>>[severity] location` with the required `>>` sentinel and severity from the closed vocabulary (`blocking`, `major`, `minor`), body prose following.
- [ ] 3.2 Verify the new prompt text stays transport-neutral — no stdin block, file path, or delimiter names.
- [ ] 3.3 Update any test asserting the baked prompt's hash or content.

## 4. Envelope and parser

- [ ] 4.1 Define the envelope in `internal/review`: `Finding{Span, Text, Location, ClaimedSeverity}` and the envelope type carrying `Raw`, `Findings`, `Status`, and `Provenance`. Document `Raw` as authoritative, `Findings` as best-effort, and the span as an envelope-local reference rather than a stable identity; add no novelty field.
- [ ] 4.2 Define `ParseStatus` with the three values `empty`, `parsed`, and `unparseable`. Do not add a `partial` value — coverage is derived from spans, per design D3.
- [ ] 4.3 Implement the parser: header matching anchored at line start, required `>>` sentinel, closed severity vocabulary, body running until the next header.
- [ ] 4.4 Assign each finding its byte span into `Raw`, starting at the header line, and assert internally that the span's slice equals the finding's text.
- [ ] 4.5 Set `Status` to `empty` when `Raw` is empty, `unparseable` when `Raw` has content and nothing parsed, `parsed` otherwise; keep partial parses.
- [ ] 4.6 Carry provenance from the provider result onto the envelope unaltered, whatever the status.
- [ ] 4.7 Add the engine-side entry point that turns a `provider.Result` into an envelope, so front-ends never parse.

## 5. Parser tests

- [ ] 5.1 Unit-test conforming output: several findings, each with severity, location, multi-line body, and a span whose slice equals the body.
- [ ] 5.2 Unit-test the status values: empty output yields `empty`; prose with no headers and output from an overridden prompt yield `unparseable`.
- [ ] 5.3 Unit-test that a reviewer writing "I found nothing" in prose yields `unparseable`, not `empty` — the collision that motivated the enum.
- [ ] 5.4 Unit-test partial parse: some findings conforming, some not — assert the conforming ones are kept, `Status` is `parsed`, and span coverage reflects the unclaimed remainder.
- [ ] 5.5 Unit-test the adversarial cases: truncation mid-finding, a bare `[severity]` line with no sentinel, an unknown severity word after the sentinel, CRLF line endings, a header with no location, wrong-cased severity (`Blocking`), a sentinel with an interleaved zero-width character, and two consecutive headers with no body between them.
- [ ] 5.5a Unit-test coverage boundaries: conforming output with no preamble reaches full span coverage; a lead-in summary before the first header is unclaimed.
- [ ] 5.6 Unit-test forged headers: a sentinel header quoted inside a finding body, and a sentinel header present in the material under review. Assert no bytes are lost and the spans show the division.
- [ ] 5.7 Assert `Raw` is preserved byte-for-byte in every case above.
- [ ] 5.8 Add the engine-level byte-exactness test: drive known bytes (CRLF, surrounding whitespace, a header-like line) through a provider into an envelope and assert `Envelope.Raw` is identical — adapter conformance cannot cover this handoff.

## 6. CLI

- [ ] 6.1 Route `cmd/second-opinion` through the envelope and print `Raw` to stdout; observable behavior must not change.
- [ ] 6.2 Confirm the stderr provenance line is unchanged and that parse status is not surfaced there — the CLI's product is raw text.
- [ ] 6.3 Run the existing CLI tests against the loopback provider to prove output is unchanged.

## 7. Documentation

- [ ] 7.1 Update the `internal/review` architecture line in `AGENTS.md` to name the envelope and the parser.
- [ ] 7.2 Add to the `AGENTS.md` testing section that whether real reviewers follow the finding-header convention is not deterministically testable and is deliberately uncovered.
- [ ] 7.3 Remove the findings-shape open question from `openspec/config.yaml`, leaving the triage and enforcement questions open.
- [ ] 7.4 Add the CI check design D1 calls for: fail if any package under `internal/provider/` imports the parser.

## 8. Verify

- [ ] 8.1 `make test` passes and `gofmt` is clean.
- [ ] 8.2 Run one real review through a provider and confirm findings parse; record the result in the change before archiving.
