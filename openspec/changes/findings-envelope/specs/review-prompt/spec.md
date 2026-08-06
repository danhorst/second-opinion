# review-prompt

## MODIFIED Requirements

### Requirement: Baked adversarial prompt
The engine SHALL ship one baked adversarial review prompt, used whenever the caller does not override it.
The prompt SHALL be transport-neutral — it MUST NOT reference any provider's material-delivery mechanism (stdin blocks, file paths, delimiter names).
The prompt SHALL request the finding-header convention the engine parses: each finding opening with the required sentinel, a severity from the closed vocabulary, and the cited location, with the finding's reasoning and proposed fix following as prose.
The requested convention MUST keep the reviewer's output readable as prose, because that text is what the CLI prints and what the envelope preserves verbatim.

#### Scenario: Same prompt across providers
- **WHEN** the same review runs through two different providers
- **THEN** both reviewers receive the same prompt text, and nothing in it presumes how the material arrived

#### Scenario: Convention is requested, not enforced
- **WHEN** the baked prompt is used
- **THEN** it asks for the finding-header convention, and a reviewer that ignores the request still produces a completed review

#### Scenario: Prompt output stays human-readable
- **WHEN** a reviewer follows the requested convention
- **THEN** its raw output reads as prose to a human, without markup a reader must decode

### Requirement: Prompt override
A caller SHALL be able to replace the baked prompt with the contents of a file.
The override replaces the prompt entirely; there is no merging.
An overridden prompt therefore does not request the finding-header convention, and the engine SHALL treat the resulting unparseable output as an ordinary parse outcome rather than a failure.

#### Scenario: Override from file
- **WHEN** the caller supplies a prompt file
- **THEN** the reviewer receives that file's contents as the prompt, and provenance's prompt identity reflects the override, not the baked prompt

#### Scenario: Override forfeits parsing, not the review
- **WHEN** a caller overrides the prompt and the reviewer's output does not follow the convention
- **THEN** the review completes normally with verbatim text preserved and the envelope's parse status set to unparseable
