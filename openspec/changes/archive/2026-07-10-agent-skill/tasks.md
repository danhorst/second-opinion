# agent-skill — tasks

## 1. Canonical skill

- [x] 1.1 Write `skills/second-opinion/SKILL.md` per design D5 (frontmatter, invocation, exit codes, provenance, neutrality rule, triage discipline)
- [x] 1.2 Add `skills/embed.go` (package `skills`, `go:embed`) and the embed-matches-repo test

## 2. Install subcommand

- [x] 2.1 Add `skill` dispatch to `main.go` and `runSkill` in `cmd/second-opinion/skill.go` with injected edges per D3
- [x] 2.2 Implement renderings per D2 (verbatim + frontmatter-stripped) and the generated stamp per D4
- [x] 2.3 Implement the antigravity import step (run when `agy` present, print otherwise, warn on failure)
- [x] 2.4 Unit tests: default install writes both files under a temp home, `--harness` selection, `--stdout` writes nothing, idempotent overwrite with an unrelated file untouched, import command constructed correctly

## 3. Roster correction

- [x] 3.1 Remove the gemini adapter from `AGENTS.md` milestones and `openspec/config.yaml` (external deps, interchangeability, locality, MCP-caller list); antigravity carries Google

## 4. Verification and wrap-up

- [x] 4.1 `make test` and `gofmt -l .` clean
- [x] 4.2 Manual: real `skill install`, confirm `~/.claude/skills/second-opinion/SKILL.md`, run/inspect `agy plugin import claude` and `agy plugin list`, `--stdout` visually checked
- [x] 4.3 Update `AGENTS.md` Architecture (skills/, skill subcommand) and add the skill milestone as done
