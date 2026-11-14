## ADDED Requirements

### Requirement: Suppress optional context and tools
Codex invocations SHALL ignore user configuration and execpolicy rules, suppress host skill discovery, and disable shell execution, code-mode host, local image reading, apps, plugins, memory, multi-agent, skill search, and web search. Authentication SHALL remain available. Unsupported controls SHALL fail rather than retry with weaker isolation.

#### Scenario: User customizations are present
- **WHEN** an invocation is constructed
- **THEN** explicit context/tool suppression controls are present in addition to isolated cwd and project-document suppression
