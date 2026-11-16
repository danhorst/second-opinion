## Context

See proposal.md. Existing adapters return raw text and provenance; the pending findings-envelope change addresses parsing separately.

## Goals / Non-Goals

Make the existing CLI usable as a predictable harness primitive. Do not add provider calls, implicit context, or a findings parser.

## Decisions

- Use a versioned CLI JSON object: `schema_version: 1`, `kind: "review"`, `raw`, and `provenance` with `provider`, `model`, `prompt_hash`. Keep the DTO in the CLI so the provider API and pending parser proposal stay independent. Errors retain the existing exit codes and stderr diagnostics, with no success JSON. Help and discovery retain their existing text output.
- `--dry-run` always emits JSON with `schema_version: 1`, `kind: "request"`, `prompt`, `material`, and `prompt_hash`. It uses the same assembly path, requires no provider, and ignores provider/model selection. It exits zero for a successful preview, never claiming a completed review. Preview exposes engine input; adapter delimiters and harness system instructions are outside this representation.
- `--timeout` accepts a strictly positive Go duration and wraps the whole provider call, so Codex fallback shares one budget. No default deadline is imposed. File assembly is synchronous; Git receives the caller context for signal cancellation. Provider process-tree cleanup remains an adapter concern.
- Harden existing argument parsing without replacing it: nonempty required values cannot consume a flag; `--` ends option parsing. Leading-dash file values can use `./` paths. Diff bases cannot start with a dash, and a final Git `--` prevents path ambiguity. Disable color, external diff and textconv because material should be the patch, not presentation or helper output.
- Check output writes. Exit 1 also covers failure to deliver a completed result, with a diagnostic that does not claim the reviewer failed to run.

- File emptiness is checked on bytes, avoiding a full-input string conversion. Keep benchmarks at 1 MiB and 10 MiB; these measure local assembly only. Add hermetic CI on PRs and main pushes using the action versions already used for releases. Real-provider conformance stays separately tagged.

## Cold-reviewer guarantee

No provider behavior changes. Preview executes no reviewer. Request assembly remains explicit and shared by preview and execution; no additional repository files or instructions reach a provider. Disabling diff helpers reduces implicit local processing.

## Risks / Trade-offs

- JSON becomes a public surface → version it and retain raw reviewer text without an inferred finding count.
- Preview and review are separate snapshots → document that files may change between calls.
- Buffers and OS argument limits still bound large reviews → document these remaining gaps; do not invent unmeasured performance gains.
- Unknown model markers cannot establish independence → teach callers this explicitly.

## Migration Plan

All controls are opt-in. Existing text callers continue to work. Reinstall the embedded skill to receive updated instructions. Reverting the change removes only the added CLI controls and hardening.
