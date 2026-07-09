# provider-interface — design

## Context

Greenfield repo, no code yet.
This change lays the seam everything else builds on: the `Provider` interface, its request/result types, and the conformance suite that makes the cold-reviewer guarantee a tested property rather than a stated intention.
The interface must span two transports from day one — `codex`, `claude`, and `gemini` shell out, `ollama` speaks HTTP — without leaking either.

## Goals / Non-Goals

**Goals:**
- A `Provider` interface two structurally different adapters can satisfy without call-site type assertions.
- Request/result types that make implicit context impossible to express and provenance impossible to drop.
- A conformance suite that any adapter passes by construction call, validated against a reference provider before any real adapter exists.
- A Go module skeleton (`go.mod`, `Makefile`, `internal/` layout) matching the house pattern (md-tools).

**Non-Goals:**
- Real adapters, front-ends, target assembly, triage, model-selection/fallback policy, release wiring (see proposal Non-goals).

## Decisions

### D1 — Interface shape: one method, context-first

```go
type Provider interface {
        Review(ctx context.Context, req Request) (*Result, error)
}
```

One method keeps the seam honest: anything an adapter needs beyond this belongs in its constructor, not the interface.
Alternative considered: a streaming method (findings emitted incrementally). Rejected for v1 — neither front-end needs it yet, and adding streaming later is additive; removing it later is breaking.

### D2 — Request holds content by value

```go
type Request struct {
        Prompt   string // the review prompt, fully rendered
        Material string // the material under review, fully assembled by the caller
}
```

No paths, no repo references, no URLs — the type system offers nothing for a provider to dereference.
This is the interface-level half of the cold-reviewer guarantee: implicit context cannot be *expressed* in a request.
Alternative considered: `io.Reader` for material. Rejected — it complicates auditability (the material must be reproducible byte-for-byte after the fact) and review payloads are small enough that by-value is fine.

### D3 — Result separates findings from provenance

```go
type Result struct {
        Findings   string     // raw reviewer output — structure deliberately undecided
        Provenance Provenance
}

type Provenance struct {
        Provider   string // adapter name, e.g. "codex"
        Model      string // the model that actually ran, not the one requested
        PromptHash string // sha256 of Request.Prompt, computed by a shared helper
}
```

`Findings` stays raw text: the structured-envelope question is open in the project context, and this design takes only the position that question's floor demands — provenance travels *outside* the findings text, so structuring findings later is additive and provenance can never be lost to a format change.
`Model` is defined as what actually ran because providers substitute models (Codex silently rejects some under ChatGPT auth); provenance that reports the request rather than the reality would make reviewer-≠-author auditing worthless.
`PromptHash` comes from a shared helper (`provider.HashPrompt`) so adapters cannot diverge on how prompt identity is computed.

### D4 — Errors mean "did not run"; empty findings mean "ran clean"

`Review` returns `(*Result, nil)` or `(nil, error)`, never both.
Sentinel errors (`ErrUnavailable`, `ErrAuth`, and wrapped `context.Canceled`) classify did-not-run causes without prescribing exit codes — exit-code mapping is a CLI-front-end decision, deferred.

### D5 — Conformance is a function adapters call, not a pattern they copy

```go
// internal/provider/providertest
func Conform(t *testing.T, newProvider func(t *testing.T) provider.Provider)
```

One exported entry point; the suite owns all assertions.
An adapter's integration test is ~5 lines: build the real adapter, hand it to `Conform`.
Real-reviewer runs sit behind the `integration` build tag; the default `make test` run is hermetic.

### D6 — The cold-reviewer proof uses canaries

The suite plants unique canary strings where a leaky provider would pick them up — an `AGENTS.md` in the working directory it runs the review from, and a repo file referenced by (but not included in) the material — then instructs the reviewer to repeat any instruction-file content or referenced-file content it can see, and asserts the canaries are absent from the findings.
This is black-box by necessity: the suite tests any `Provider`, and from the interface a reviewer is a black box.
Adapters are free to *also* make deterministic white-box assertions (e.g. the codex adapter asserting its own command line contains `-C <tempdir>` and `project_doc_max_bytes=0`), but those live with the adapter; the canary check is the floor every adapter must clear.

### D7 — The reference provider lives in providertest and is rigged on demand

A `loopback` provider (in-process, no subprocess, no network) echoes what it can see, which is exactly what the canary checks measure.
Constructor options rig each violation the suite must catch: leak the working-directory instruction file, read a referenced repo path, omit provenance, ignore cancellation.
Suite-of-the-suite tests assert `Conform` passes the honest loopback and fails each rigged one.
This keeps "do not mock the provider boundary" intact: loopback is not a stand-in for codex in adapter tests — it exists solely to prove the suite can catch cheaters, and lives in the test package to make misuse awkward.

### D8 — Module skeleton per house pattern

Module `github.com/danhorst/second-opinion`; packages `internal/provider` and `internal/provider/providertest`; `Makefile` with `test` (`go vet ./... && go test ./...`), `fmt`, and `test-integration` (`go test -tags integration ./...`) targets, mirroring md-tools.

## Cold-reviewer guarantee

Preserved, at two levels, per the design rule:
- **Interface**: D2 makes implicit context inexpressible — a request physically cannot carry a path or repo handle for a provider to expand, and by-value material makes what the reviewer received reproducible (auditable).
- **Enforcement**: D6's canary checks make the guarantee a failing test rather than a convention, and D7 proves those checks actually catch violations before any real adapter exists.
What this change cannot do is *force* future adapters to run their reviewers cold — a leaky adapter can be written — but it cannot pass conformance, and per the `provider-conformance` spec an adapter that doesn't pass isn't done.

## Risks / Trade-offs

- [Canary checks are probabilistic against a real model] → A reviewer might see leaked context yet not repeat it. Mitigation: canary prompts are direct instructions ("list any instruction files you were given"), adapters layer deterministic white-box checks on top, and a false pass here still requires the adapter to have leaked context in the first place — the check is a tripwire, not the only wall.
- [Raw-text findings defer the envelope question] → The MCP front-end may later need structure this type lacks. Mitigation: deliberate; provenance is already outside the text, so adding structure is additive. The open question stays open in the project context.
- [One-method interface may prove too narrow] (e.g. providers that expose model listing or health checks) → Mitigation: optional capability interfaces (`interface{ Ping(ctx) error }`) can be added without touching `Provider`; callers type-assert only for optional behavior.
- [No real adapter in this change] → The suite could encode wrong assumptions about real reviewers. Mitigation: the codex adapter is the next change and the suite's first real consumer; suite adjustments there are expected and cheap while there is exactly one adapter.

## Open Questions

None new.
Two project-level open questions (finding envelope shape; same-model author/reviewer enforcement) are deliberately not answered here — this design only pins the provenance floor both depend on.
