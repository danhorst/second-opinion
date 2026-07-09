@AGENTS.md

`AGENTS.md` is read by every agent that runs here, `codex` included — it loads the file automatically from its working directory.
Keep it to facts about the project.
Claude-harness behavior belongs in this file, which `codex` never reads.

## Delegate down

Cost per token is a design constraint, not an afterthought.
Reach for the cheapest model that can hold the work, and escalate only on evidence that it could not.

- Haiku — file surveys, mechanical edits, running tests, gathering output.
- Sonnet — implementation against a spec that is already settled.
- Opus — design, spec authoring, adversarial review, and any judgment about novelty.
- Fable — the hardest long-horizon design and review work, where a wrong answer costs more than 2x Opus pricing does.

Cheap is not free above a certain size.
A cheap implementer on a large task converges on the tests and degrades on design, and the cost of specifying the work for it exceeds what one capable model would have charged to do it.
Past that point, go direct to the most capable model rather than decomposing.

The driving model does not set the ceiling.
When a cheaper model drives the session, design and spec work still escalate rather than being absorbed — a cheap spec is expensive twice.

These tiers are an ordering, not a price list.
Check current pricing before optimizing against it.
