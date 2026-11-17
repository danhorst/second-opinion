---
name: second-opinion
description: Get an adversarial second opinion on a spec, design doc, or diff from a model that did not write it, using the second-opinion CLI or stateless MCP tools. The external model finds; you filter.
---

Get an adversarial second opinion on a spec, design doc, or diff from a model that did not write it.
The reviewer finds; you filter.
Raw reviewer output is mostly triage overhead — the value is in the filtering, which is your job, not the tool's.

## Run it

`second-opinion` owns the orchestration — target assembly, cold invocation, and provider transport.

- A document: `second-opinion --provider P path/to/SPEC.md [more.md ...]`
- The current diff: `second-opinion --provider P --diff [BASE]` (BASE defaults to `HEAD`)
- Override the prompt: `--prompt-file prompt.txt`. Force a model: `--model M`.
- For harness consumption: add `--json` to get one versioned object with `raw` and `provenance` (`provider`, `model`, full `prompt_hash` and `material_hash`). This does not parse or rank findings.
- Inspect input first: `second-opinion --dry-run --diff HEAD` or `--dry-run path/to/SPEC.md`. The JSON contains the exact engine `prompt` and `material`; no provider is needed or constructed, and provider/model selection is ignored. Keep inputs unchanged before the actual review.
- Bound reviewer latency: `--timeout 2m`. This covers the provider call including fallbacks, after assembly; omission means unlimited. On macOS/Linux/BSD the invocation process group is terminated; detached children are outside this guarantee, and other platforms currently cancel only the parent.
- Literal filenames: use `--` before paths beginning with a dash.

Diff mode includes tracked staged and unstaged changes against the base, but excludes untracked files. Select new files explicitly in file mode. Supporting files must be deliberately bundled; the reviewer cannot be expected to fetch missing context.

**Choose the reviewer, don't default it.**
The tool ships no default provider — pass `--provider` or set `SECOND_OPINION_PROVIDER`.
Pick a reviewer that differs from the model that wrote the material: a different model of the same family is a valid second opinion; the same model defeats the premise.
Run `second-opinion --list-providers` to see what is registered.

**Model forcing is provider-specific.**
Some providers reject or substitute forced models depending on their auth; the tool handles the known fallbacks.
Read the `reviewed-by:` line or JSON provenance before trusting the requested model.
`codex-default`, `antigravity-default`, and `claude-default` mean the model identity is unknown.
Codex uses reported session model metadata when present, with a requested-name/default-marker fallback. Antigravity forced names are requested identities without independent model metadata; Claude uses usage metadata when present, with a requested-name/default-marker fallback.
Report uncertainty when model independence cannot be established; a different provider alone does not prove a different model.

**Read the output channels.**
Text mode puts findings on stdout. JSON mode puts raw provider-returned text and provenance in one object on stdout; it adds no finding count or severity judgment.
One provenance line also arrives on stderr — `reviewed-by: provider=… model=… prompt=…` — subject to the identity limitations above.
Check it before triaging: if the model matches the material's author, the review is self-review and the findings are suspect.

**Exit codes.**
0 means the requested operation succeeded: a review completed, or preview/help/discovery succeeded. A preview is not a completed review.
1 means review or output failure (cause on stderr: binary missing, auth, usage limit, nothing to review, timeout, or output write failure).
2 means the invocation was invalid.
Check status before parsing JSON; a failed operation does not emit a success object. Help remains text. `--list-providers --json` returns registered names and their `provider_isolation` metadata without checking credentials.

To require a local client material boundary, pass `--require-isolation local-material-only` (MCP: `require_isolation`). Unsupported adapters fail before review without fallback. The HTTP adapter supports it; harness adapters report `unverified`. Successful results carry effective `isolation`. The configured endpoint and transport remain trusted: this is not endpoint context or OS isolation certification. Dry-run records the requirement without claiming it is satisfied.

## MCP calls

Run `second-opinion mcp serve` as a stdio MCP server. For private HTTP setup and token handling see the repository README; the endpoint is loopback-only and requires a bearer token on every request.

- Discover with `list_providers({})`; registered names are not an availability or authentication guarantee.
- Call `review` with explicit `provider` and complete `material`. Optional fields are `model`, `prompt` (full replacement), `require_isolation` (`local-material-only`), and integer `timeout_ms` (1–300000, default 120000). Environment provider defaults do not apply.
- Assemble the patch/document and deliberately selected supporting context yourself. Paths and URLs remain text; no workspace reads, roots, previous context, stored results, or result retrieval exist.
- Preserve `raw`, full `provenance`, and the request receipt. Structured content and the single JSON text block contain the same versioned outcome. No parsed finding list is required. Unknown model markers do not prove reviewer independence.
- Check `isError` and `kind` before triage. A completed empty review is success. Errors expose a safe `code` and execution status: input errors, `busy`, and `isolation_unsupported` are `not_started`; other execution failures are `unknown`.
- Treat lost/cancelled/timed-out responses as unknown completion. Retrying may repeat paid work; request IDs are not idempotency keys. There is no recovery endpoint or automatic gateway retry.
- Respect discovery limits: 1 MiB combined UTF-8 material/prompt, 256-byte model, two simultaneous reviews, and bounded transport/output budgets. An overflow fails rather than returning truncated findings. `busy` is not queued work.

## Triage the output

The reviewer returns many findings, ranked by its own sense of severity, and it does not always read the whole document.
Do not relay the raw dump.
Filter:

1. **Cluster.** Collapse findings that circle one root cause into a single item.
2. **Dedupe against the document.** Drop anything that restates a decision the document already pinned — the reviewer flagging your own stated constraint as "blocking" is noise.
3. **Rank by severity × novelty.** A "blocking" that repeats a pinned decision is noise; a "minor" you had not seen is signal. Novelty is the axis the reviewer cannot judge and you can.
4. **Split by altitude.** Separate findings that belong in the document (fold into the spec) from implementation-level ones (defer to the code).
5. **Name the spec-gaps.** A finding that the *document* failed to require something grades the document, not the author — surface it as a gap to close, not a defect.

## Report

Present a synthesis, not the transcript:

- The one or few findings that genuinely matter, with the section each cites.
- The cheap, real fixes worth taking regardless.
- What was noise or already-covered, named briefly so the reader can see the filter you applied.
- The `reviewed-by:` line, so the reader knows who reviewed.

Then act on the reader's call.
Many reviews contain repetition or already-covered concerns. Filter based on evidence, with no rejection quota: keep every supported finding even when most survive.
