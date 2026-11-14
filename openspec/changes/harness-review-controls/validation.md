# Validation — 2026-11-13

- `make test`: passed (vet and all default tests).
- `go test -race ./...`: passed.
- `go test -tags integration -run '^$' ./...`: all integration-tagged packages compiled; no live tests executed.
- CLI build to `/private/tmp/second-opinion-harness-review`: passed.
- Built-binary smoke checks: file and diff previews decoded as JSON with the expected kind, schema, material and prompt hash; stderr empty. Discovery and invalid-timeout exit status passed.
- `openspec validate --all --strict`: all 10 current specs/changes passed after syncing the delta specs.
- `git diff --check` and `gofmt -l cmd internal skills`: clean.
- Assembly benchmark ran twice before and after the change. Measurements and limits are in `docs/reviews/2026-11-13.md`.

Tests used `GOCACHE=/private/tmp/second-opinion-go-cache` because the sandbox cannot write the user's default build cache. The clean build also used `GOMODCACHE=/private/tmp/second-opinion-mod-cache`; no dependency was added. An intermediate skill-content test expected the removed “two-thirds” quota; its assertion now checks the replacement “no rejection quota” guidance, and the final suite passes.

No real-provider calls were made. Adapter behavior and real-binary conformance were not re-certified. Hosted CI was added but cannot be reported as run locally. The pre-existing Codex edits and archive move remain untouched. The findings-envelope proposal remains pending.

The `review-cli` and `agent-skill` specs were synced; this completed change is retained unarchived for inspection.
