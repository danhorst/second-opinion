# review-cli — design

## Context

First front-end over a proven seam: three conformant adapters behind one interface, a contract whose provenance and error semantics were designed for exactly this consumer.
The predecessor script (`second-opinion.sh`) supplies the UX to preserve — `PATH...` or `--diff [BASE]`, `-p` prompt override, model forcing — and the bug to bury (reviewer runs in the caller's repo).

## Goals / Non-Goals

**Goals:**
- `second-opinion [flags] PATH...` / `second-opinion --diff [BASE]` working end-to-end through any registered provider.
- Target assembly and the baked prompt in `internal/review`, reusable by the MCP front-end unchanged.
- The deferred surface decisions made and specced: exit codes, output channels, provider selection.

**Non-Goals:** see proposal — MCP, triage, dotfiles retirement, release wiring, enforcement.

## Decisions

### D1 — Package split: `internal/review` engine-side, `cmd/second-opinion` thin

`internal/review` owns what both front-ends need: the baked prompt, target assembly (`FromFiles`, `FromDiff`), and the provider registry.
`cmd/second-opinion` owns only flag parsing, wiring, and exit codes.
The MCP change should need zero edits to `internal/review` for its core path — that's the test of this split.

### D2 — Provider registry: a map, not a plugin system

```go
// internal/review
func Providers() []string
func NewProvider(name, model string) (provider.Provider, error)
```

A hand-maintained map from name to constructor (`codex`, `claude`, `antigravity`).
Adding an adapter means adding one line — acceptable coupling for an internal registry; anything more is machinery without a second consumer.

### D3 — Selection: flag, then env, then refuse

`--provider` wins; `SECOND_OPINION_PROVIDER` is the ambient default for daily use; with neither the CLI refuses with the provider list.
Refusal is the neutrality decision: a baked default would make one vendor the tool's home provider — the exact bias the project context forbids.
Alternative considered: default to "first installed binary found." Rejected — silently varying behavior by machine state, and an implicit vendor preference in the probe order.

### D4 — Target assembly mirrors the predecessor, with headers

`FromFiles(paths)` concatenates in argument order, each file preceded by `=== path ===` so multi-file reviews cite locations; `FromDiff(base)` runs `git diff <base>` in the caller's cwd (caller-side assembly is explicit assembly — the cold guarantee constrains the *reviewer's* context, not the caller's).
Empty material is refused before any provider is constructed: a review of nothing burns money to say nothing.

### D5 — Baked prompt: predecessor's content, neutral wording

The prompt migrates verbatim except its first paragraph, where "follows in the `<stdin>` block" becomes "follows this prompt" — the only phrase that presumed a transport.
Stored as a Go string constant; `--prompt-file` replaces it wholesale; `provider.HashPrompt` (already the provenance mechanism) identifies whichever ran.

### D6 — Output and exit codes

Findings verbatim on stdout; `reviewed-by: provider=%s model=%s prompt=%s` on stderr.
Exit 0 = review completed (a clean review is a result, not an error — this tool's findings are read by agents and humans, not gated in shell conditionals like grep); exit 1 = did not run (message carries the sentinel classification); exit 2 = usage.
Fixed here after being deliberately deferred through two changes; the spec makes it a compatibility promise.

### D7 — Testing

Assembly, registry, selection, and prompt composition are pure and unit-tested.
The wiring test drives the built binary against the `Loopback` reference provider — registered only in test builds — rather than mocking anything: the CLI's job is wiring, and the loopback exists precisely to let wiring be exercised without a network.
A small integration-tagged smoke test runs one real review end-to-end through one provider.

## Cold-reviewer guarantee

Unaffected at the mechanism level — the CLI builds material caller-side and hands it to adapters whose cold posture is already conformance-proven.
The guarantee-relevant decision is D4's framing: assembly in the caller's cwd is the *caller's explicit act*, which is precisely what the nothing-implicit invariant permits; the reviewer still sees only what was assembled.

## Risks / Trade-offs

- [No default provider adds first-run friction] → One env var, set once; the error message says exactly what to do. Neutrality is worth one line of setup.
- [Exit-0-on-findings differs from grep-style tools] → Deliberate and specced; agents consuming the CLI read findings, not exit codes. Revisiting later would be a breaking change, hence decided in a spec, not a README.
- [`git diff` shells out from the CLI] → The one place the front-end touches the caller's repo; it is read-only and caller-initiated. Failure (not a repo, bad base) maps to exit 2 with git's message.

## Open Questions

None new. The envelope question is untouched: stdout carries raw findings, so structuring later is additive (a `--format json` that wraps findings + provenance).
