---
name: second-opinion
description: Get an adversarial second opinion on a spec, design doc, or diff from a model that did not write it, using the second-opinion CLI. The external model finds; you filter.
---

Get an adversarial second opinion on a spec, design doc, or diff from a model that did not write it.
The reviewer finds; you filter.
Raw reviewer output is mostly triage overhead — the value is in the filtering, which is your job, not the tool's.

## Run it

`second-opinion` owns the orchestration — target assembly, cold invocation, and provider transport.

- A document: `second-opinion --provider P path/to/SPEC.md [more.md ...]`
- The current diff: `second-opinion --provider P --diff [BASE]` (BASE defaults to `HEAD`)
- Override the prompt: `--prompt-file prompt.txt`. Force a model: `--model M`.

**Choose the reviewer, don't default it.**
The tool ships no default provider — pass `--provider` or set `SECOND_OPINION_PROVIDER`.
Pick a reviewer that differs from the model that wrote the material: a different model of the same family is a valid second opinion; the same model defeats the premise.
Run without a provider to see what is registered.

**Model forcing is provider-specific.**
Some providers reject or substitute forced models depending on their auth; the tool handles the known fallbacks.
Trust the `reviewed-by:` line, not the flag you passed.

**Read the output channels.**
Findings arrive verbatim on stdout.
One provenance line arrives on stderr — `reviewed-by: provider=… model=… prompt=…` — and it reports what *actually* ran.
Check it before triaging: if the model matches the material's author, the review is self-review and the findings are suspect.

**Exit codes.**
0 means the review completed — findings, including none, are data.
1 means the reviewer did not run (cause on stderr: binary missing, auth, usage limit, nothing to review).
2 means the invocation was invalid.

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
Expect roughly two-thirds of the raw output to fall away in triage; if it does not, you are under-filtering.
