## ADDED Requirements

### Requirement: Harness controls and provenance limitations
The canonical skill SHALL document JSON results, request preview, review timeout, literal path arguments, and the exclusion of untracked files from diff mode. It SHALL explain that default-model markers and requested model names without provider verification do not prove the reviewer differs from the author.

#### Scenario: Caller cannot identify the reviewer model
- **WHEN** a result carries a default-model marker
- **THEN** the skill directs the caller to report the uncertainty rather than claiming verified model independence

## MODIFIED Requirements

### Requirement: Skill content covers invocation and triage
The skill SHALL teach a calling agent: how to invoke the CLI (targets, provider selection, model forcing, prompt override), how to interpret exit codes and the provenance line, that the reviewer should differ from the material's author model, and the triage discipline — cluster, dedupe against the document, rank by severity times novelty, split by altitude, name the spec-gaps — filtering based on evidence without a fixed rejection quota.
The content MUST be provider-neutral: no provider is presented as the default or home reviewer.

#### Scenario: Provider-neutral wording
- **WHEN** the skill names providers
- **THEN** it presents them as a set the agent chooses from, never one as the tool's default
