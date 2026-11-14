## MODIFIED Requirements

### Requirement: Per-harness installation
`second-opinion skill install` SHALL write the skill natively for Claude Code (`~/.claude/skills/second-opinion/SKILL.md`, verbatim) and Codex (`~/.agents/skills/second-opinion/SKILL.md`, with canonical frontmatter), defaulting to all targets, selectable via `--harness`.
For Antigravity, which consumes Claude-format plugins (not bare skills), the installer SHALL write a minimal plugin (`plugin.json` plus the skill) under `~/.second-opinion/antigravity-plugin` and run `agy plugin install <dir>` when `agy` is on PATH, printing the command as a next step when it is not.
`--stdout` SHALL print the canonical skill instead of installing, for any other harness.

#### Scenario: Default install reaches all harnesses
- **WHEN** `skill install` runs with no flags
- **THEN** the Claude and Codex files are written, the Antigravity plugin directory is written, and the plugin install runs or is printed as a next step

#### Scenario: Stdout escape hatch
- **WHEN** `skill install --stdout` runs
- **THEN** the canonical skill is printed to stdout and no file is written

#### Scenario: Existing legacy prompt is preserved
- **WHEN** installation runs while `~/.codex/prompts/second-opinion.md` already exists
- **THEN** it writes the native skill without changing or deleting the legacy prompt
