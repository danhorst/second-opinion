# antigravity-adapter — design

## Context

Second real adapter, same seam. `agy` differs from codex on every axis the adapter layer must absorb: no working-directory flag (isolation comes from the process's cwd), no stdin in print mode (material rides in the prompt argument), no structured output (provenance degrades honestly), and it is itself a multi-model harness (Gemini, Claude, GPT-OSS behind one binary).
Empirical facts pinned by probe on 2026-07-10: `agy --print` works under the installed auth; piped stdin is not visible to the model.

## Goals / Non-Goals

**Goals:**
- `internal/provider/antigravity` passing full conformance against the real binary — the suite's first complete real-binary verification.
- Cold invocation with deterministic white-box tests.
- Honest provenance under a harness that doesn't report what ran.

**Non-Goals:** see proposal.

## Decisions

### D1 — Invocation: cwd isolation, sandbox on, composed instruction

```
agy --print --sandbox [--model <model>] <instruction>
```

executed with `cmd.Dir` set to a fresh empty temp directory (agy has no `-C`; the process working directory is the isolation mechanism, same effect as codex's flag).
No `--add-dir` — the workspace is exactly the empty directory.
No `--dangerously-skip-permissions` — a review needs no tools; if the harness wants a permission mid-review, failing is correct.
`buildInvocation(req, model, workdir)` stays a pure function returning argv; there is no stdin.

### D2 — Material embedded via a pure composer

```go
func composeInstruction(req provider.Request) string
```

Prompt first, then the material fenced in a delimited block:

```
<prompt>

--- MATERIAL (verbatim, assembled by the caller) ---
<material>
--- END MATERIAL ---
```

Deterministic, so what the reviewer received is reproducible from the request — the auditability requirement holds even though the transport differs from codex's stdin.
Trade-off: argv carries the whole instruction, so ARG_MAX (~1MB on macOS) bounds material size; typical diffs and documents fit, and oversized material fails loudly at exec. Accepted for v1; noted in risks.

### D3 — Provenance: forced model or documented marker

`agy` prints only the response text — nothing structured names the model that ran.
Forced model (`WithModel`, passed as `--model`) → provenance names it; agy either honors the name or errors, so the report is truthful.
No forced model → `Model: "antigravity-default"` — the same documented-degradation pattern the codex design allows, here as the *normal* unforced path.
This is exactly the provenance floor the project's open questions need to stay honest: a marker that says "unknown default," never an invented model name.

### D4 — Error classification mirrors codex's shape

`exec.ErrNotFound` → `ErrUnavailable`; stderr/stdout matching sign-in or authentication signatures → `ErrAuth`; anything else wrapped with the first meaningful diagnostic line.
Signatures live in constants; unit-tested as pure predicates.
No model-fallback logic: agy has no known auth-dependent model rejection (unlike codex's ChatGPT-auth behavior); if one surfaces in integration, it gets the codex treatment then.

### D5 — Constructor shape mirrors codex

`antigravity.New(opts ...Option) *Provider`, `WithModel(string)`, provider name constant `"antigravity"`, binary name `"agy"` fixed.
The two adapters deliberately rhyme — a third (`claude`) should be able to copy either.

## Cold-reviewer guarantee

Preserved by construction and proven the same two ways as codex:
- **Mechanism (D1):** empty-tempdir cwd closes instruction-file auto-loading from the caller's directory; no `--add-dir` closes workspace access; `--sandbox` restricts what the harness's tools can do; material arrives by value inside the instruction.
- **White-box proof:** unit tests on `buildInvocation`'s argv and cwd.
- **Black-box proof:** the conformance canaries (all three instruction files, plus the repo-read check) against the real binary — and unlike codex, this run isn't quota-blocked, so this adapter is where the suite's canary checks meet a real model for the first time in full.

## Risks / Trade-offs

- [ARG_MAX bounds material size] → ~1MB on macOS; typical review payloads fit. Oversized material fails loudly at exec with a clear error. Revisit (temp-file handoff inside the adapter's own workdir) only if real usage hits it.
- [`antigravity-default` weakens same-model detection for unforced reviews] → Documented, not silent; callers who care about the reviewer-≠-author audit force a model. The open question's eventual answer can make forcing mandatory.
- [`--sandbox` semantics are underdocumented] → The canary checks measure the property we actually need (no caller-context leakage) rather than trusting the flag's description.
- [Print-mode timeout defaults to 5m inside agy] → Context cancellation still kills the process from our side; the double timeout is harmless.

## Open Questions

None new.
