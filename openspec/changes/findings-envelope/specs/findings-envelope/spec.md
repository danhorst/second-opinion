# findings-envelope

## ADDED Requirements

### Requirement: The envelope carries verbatim reviewer text
The engine SHALL produce, for every completed review, an envelope holding the reviewer's output verbatim.
That verbatim text is the authoritative record of what the reviewer said, and it MUST be populated whenever the reviewer ran — including when the reviewer found nothing and when parsing recovered no findings.

#### Scenario: Verbatim text always present
- **WHEN** a review completes through any provider
- **THEN** the envelope holds the reviewer's output byte-for-byte as the provider returned it

#### Scenario: Empty review
- **WHEN** the reviewer runs and produces no output
- **THEN** the envelope holds empty verbatim text, an empty finding list, and a parse status of empty, and the review is still a completed review

### Requirement: Verbatim text is byte-exact through the engine boundary
The envelope's verbatim text SHALL be byte-identical to what the provider returned, across the provider-to-engine handoff and envelope construction.
Verifying this at the adapter alone is insufficient, because an adapter-level check cannot observe mutation introduced after the provider returns.

#### Scenario: Bytes survive envelope construction
- **WHEN** a provider returns output containing CRLF line endings, leading and trailing whitespace, and a line resembling a finding header
- **THEN** the envelope's verbatim text is identical to that output, with no normalization, trimming, or re-wrapping

### Requirement: Parse status distinguishes the outcomes that occur
The envelope SHALL carry a parse status with at least three distinct values: empty (the reviewer produced nothing to parse), parsed (at least one finding was recovered), and unparseable (the verbatim text has content and no finding was recovered).
A two-valued flag MUST NOT be used, because it conflates empty with unparseable.

#### Scenario: Reviewer reports no findings in prose
- **WHEN** the reviewer runs and writes a sentence stating it found nothing, in no particular format
- **THEN** the status is unparseable rather than empty, and the verbatim text carries the sentence

#### Scenario: Empty and unparseable are distinguishable
- **WHEN** a consumer reads the parse status
- **THEN** it can tell a reviewer that produced nothing from a reviewer whose output the parser could not decompose

#### Scenario: Unparseable does not convey reviewer intent
- **WHEN** the status is unparseable
- **THEN** the consumer MUST NOT infer whether the reviewer reported no findings or produced output the parser failed on, because the value deliberately covers both; the verbatim text is where that distinction is legible

### Requirement: Findings are parsed best-effort and are not authoritative
The engine SHALL parse the verbatim text into a list of individual findings on a best-effort basis.
The list MUST NOT be treated as a faithful decomposition of the verbatim text: it MAY contain fewer findings than the reviewer wrote, and consumers SHALL treat the verbatim text as the record of what was said.

#### Scenario: Partial parse is kept
- **WHEN** some of the reviewer's findings match the convention and others do not
- **THEN** the envelope holds the findings that matched, the status is parsed, and the verbatim text still holds all of them

#### Scenario: Count is not a claim
- **WHEN** a consumer reads the finding list
- **THEN** the envelope's documentation states the list is best-effort, so the list length is not presentable as the number of findings the reviewer reported

### Requirement: A finding's address is its byte span
Each parsed finding SHALL carry the byte span it occupies within the verbatim text, and the finding's text MUST equal the verbatim text over that span.
The span SHALL begin at the finding's header line, not at the first line of its body, so that fully conforming output leaves no header bytes unclaimed.
The span SHALL be an envelope-local reference: it is deterministic and unique within one envelope, and it MUST NOT be presented as an identity stable across envelopes, reviews, or parser versions.

#### Scenario: Span addresses a finding
- **WHEN** a consumer holds a parsed finding
- **THEN** it can refer to that finding by a span that indexes the verbatim text, without relying on array position

#### Scenario: Span content matches the record
- **WHEN** a finding's span is applied to the verbatim text
- **THEN** the result equals the finding's text exactly, header line included

#### Scenario: Conforming output reaches full coverage
- **WHEN** reviewer output consists entirely of conforming findings with no preamble
- **THEN** the spans account for all of the verbatim text, leaving no bytes unclaimed

#### Scenario: Coverage is derivable
- **WHEN** a consumer needs to know how much of the reviewer's output the finding list accounts for
- **THEN** it can compute that from the spans, because the engine SHALL NOT store a separate partial-parse claim it cannot honestly compute

#### Scenario: Preamble is unclaimed
- **WHEN** the reviewer writes a lead-in summary before the first header line
- **THEN** that text falls outside every span and counts as unclaimed on the same terms as any other unmatched text, with no distinct category

### Requirement: Finding-header convention
The baked prompt SHALL request, and the parser SHALL recognize, a line-oriented convention: each finding opens with a header line carrying a required sentinel, a severity from a closed vocabulary, and the location the reviewer cites; everything until the next header line is that finding's body.
The parser SHALL anchor header matching at the start of a line and SHALL require both the sentinel and the closed severity vocabulary.
Severity matching SHALL be case-sensitive, and the sentinel and severity token SHALL be plain ASCII: a header candidate carrying interleaved zero-width or control characters MUST NOT match, so that a line cannot parse as a header while reading as ordinary prose to a human, or the reverse.

#### Scenario: Conforming output parses into findings
- **WHEN** reviewer output contains several findings written in the convention
- **THEN** the envelope holds one finding per header line, each carrying its severity, its cited location, its body, and its span

#### Scenario: A bare severity bracket does not open a finding
- **WHEN** a line begins with a bracketed severity but lacks the sentinel
- **THEN** it does not start a new finding and remains part of the current finding's body

#### Scenario: Unknown severity word does not open a finding
- **WHEN** a line begins with the sentinel followed by a word outside the severity vocabulary
- **THEN** it does not start a new finding, and the line remains part of the current finding's body and of the verbatim text

#### Scenario: Missing location
- **WHEN** a finding's header carries the sentinel and a severity but cites no location
- **THEN** the finding parses with an empty location rather than being discarded

#### Scenario: Severity casing is a near miss, not a match
- **WHEN** a header carries the sentinel followed by a severity in different casing, such as `Blocking` or `BLOCKING`
- **THEN** it does not open a finding, and the line remains part of the current finding's body

#### Scenario: Obfuscated sentinel does not match
- **WHEN** a header candidate carries zero-width or control characters interleaved with the sentinel or severity token
- **THEN** it does not open a finding

#### Scenario: Consecutive headers yield an empty body
- **WHEN** two header lines appear with no text between them
- **THEN** the first parses as a finding with an empty body rather than being merged into the second or discarded

### Requirement: Forged headers are contained, not prevented
A forged header MUST NOT lose content: the verbatim text SHALL remain complete and authoritative, and the spans SHALL make the resulting division inspectable.
The material under review is caller-supplied and may contain text resembling the convention, including when the material documents the convention itself, so the engine SHALL NOT claim to prevent a reviewer from emitting a header-shaped line it quoted from the material.

#### Scenario: Reviewer quotes a header from the material
- **WHEN** the material contains a line matching the convention and the reviewer quotes it inside a finding's body
- **THEN** the quoted line may open a spurious finding, and the verbatim text still contains every byte the reviewer wrote

#### Scenario: A forged header reassigns a body
- **WHEN** a spurious header appears mid-body, so the following prose is attributed to it rather than to the finding that introduced it
- **THEN** no bytes are lost, and a consumer inspecting the spans can see exactly how the text was divided

### Requirement: Unparseable output is not failure
Reviewer output that yields no parsed findings SHALL NOT be an error.
The engine SHALL set the status accordingly, keep the verbatim text, and leave the review's success and the front-end's exit status unchanged.

#### Scenario: Overridden prompt yields unparseable output
- **WHEN** a caller replaces the baked prompt, so the reviewer is never asked for the convention
- **THEN** the review completes, the envelope holds the verbatim text with no findings and a status of unparseable, and the exit status is that of a completed review

### Requirement: Severity is the reviewer's claim
A parsed finding SHALL carry the reviewer's severity under a name that identifies it as the reviewer's own claim rather than an engine judgment.

#### Scenario: Severity is attributed
- **WHEN** a consumer reads a finding's severity
- **THEN** the field's name and documentation attribute it to the reviewer, consistent with the guarantee that the reviewer finds but does not rank

### Requirement: No novelty field
The envelope SHALL NOT carry a novelty field on findings.
The engine cannot compute novelty, and an empty field invites the caller — frequently the model that wrote the material — to supply it.

#### Scenario: Novelty is absent from the type
- **WHEN** a consumer inspects the finding type
- **THEN** there is no novelty field to populate

### Requirement: Provenance survives the envelope
The envelope SHALL carry the review's provenance — provider, model that actually ran, and prompt identity — unaltered from the provider result.
Provenance MUST survive regardless of the parse status.

#### Scenario: Provenance on an unparseable envelope
- **WHEN** parsing recovers no findings
- **THEN** the envelope still names the provider, the model that ran, and the prompt identity

### Requirement: Parsing is engine-side
Parsing SHALL happen once, in the engine, on output already returned by a provider.
No adapter SHALL parse, restructure, or reformat reviewer output.

#### Scenario: A new adapter needs no parsing code
- **WHEN** a new provider adapter is added
- **THEN** it returns raw reviewer text and provenance, and gains findings parsing without implementing any
