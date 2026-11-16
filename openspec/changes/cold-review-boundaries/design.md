## Context

The relative-path check can pass solely because cwd changed. Installed help was inspected on 2026-11-13: codex-cli 0.167.0-alpha.4.1 and Claude Code 2.1.234. The official Codex configuration reference documents shell and image tool toggles; installed feature listing additionally identifies host-skill discovery suppression.

## Goals / Non-Goals

Reduce implicit context, prove a stronger observable cold boundary, and remove Claude's material argument limit. Do not imply model obedience proves exhaustive access denial or that these controls override administrator-managed policies.

## Decisions

Keep the relative-path case and add an independent absolute-path canary after the same positive-control gate. Validate nil/error results instead of panicking in checks. Active cancellation is tested using actual process readiness in the shared lifecycle helper: the Provider interface has no readiness signal and a prompt requesting long work cannot prove the reviewer was active.

Codex uses --ignore-user-config and --ignore-rules while retaining the original authentication home. Disable shell, code-mode host, view-image, apps, plugins, memories, multi-agent and skill search, enable skip_host_skill_discovery, and disable web search. Use strict config so unsupported keys fail rather than silently weaken isolation. Keep empty cwd and project-doc suppression.

Claude uses --safe-mode, --disable-slash-commands, --strict-mcp-config, and --no-session-persistence alongside empty tools and setting sources. Safe mode retains normal authentication, unlike bare mode. Deliver the deterministic instruction on stdin with no positional prompt. Real positive control must validate this transport.

Codex skill installation writes frontmatter-preserving content to `~/.agents/skills/second-opinion/SKILL.md`, the current documented user skill root. Existing `~/.codex/prompts/second-opinion.md` files remain untouched; no destructive migration or actual user-home install is performed during development.

## Cold-reviewer guarantee

The changes remove context and capabilities; they add no content. Absolute paths in test material are explicit pointers the reviewer must not dereference. Administrator policy, hidden provider prompts, and undeclared environment behavior cannot be exhaustively proved absent by these checks. Record observed failures honestly.

## Risks / Trade-offs

Older harness versions can reject new controls and fail closed. Codex ignores user model defaults together with user instructions; callers should force a model explicitly when needed. Antigravity has no observed equivalent to complete tool/global-context suppression, so its historical pass is not evidence for the stronger check.

## Evidence

- https://learn.chatgpt.com/docs/config-file/config-reference (shell_tool, view_image, web_search)
- https://learn.chatgpt.com/docs/build-skills (native user skill directory)
- Local `codex exec --help`, `codex features list`, `claude --help`, `agy --help`.
