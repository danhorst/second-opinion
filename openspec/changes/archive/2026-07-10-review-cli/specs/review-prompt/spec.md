# review-prompt

## ADDED Requirements

### Requirement: Baked adversarial prompt
The engine SHALL ship one baked adversarial review prompt, used whenever the caller does not override it.
The prompt SHALL be transport-neutral — it MUST NOT reference any provider's material-delivery mechanism (stdin blocks, file paths, delimiter names).

#### Scenario: Same prompt across providers
- **WHEN** the same review runs through two different providers
- **THEN** both reviewers receive the same prompt text, and nothing in it presumes how the material arrived

### Requirement: Prompt override
A caller SHALL be able to replace the baked prompt with the contents of a file.
The override replaces the prompt entirely; there is no merging.

#### Scenario: Override from file
- **WHEN** the caller supplies a prompt file
- **THEN** the reviewer receives that file's contents as the prompt, and provenance's prompt identity reflects the override, not the baked prompt

### Requirement: Prompt identity in provenance
The prompt that actually ran — baked or overridden — SHALL be identified in the result's provenance via the contract's prompt hash.

#### Scenario: Distinguishable prompts
- **WHEN** two reviews run with different prompts
- **THEN** their provenance carries different prompt hashes
