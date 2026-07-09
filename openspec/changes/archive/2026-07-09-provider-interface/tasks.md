# provider-interface — tasks

## 1. Module skeleton

- [x] 1.1 Initialize `go.mod` as `github.com/danhorst/second-opinion` and pin the Go version
- [x] 1.2 Add `Makefile` with `test` (`go vet ./... && go test ./...`), `fmt` (`gofmt -w .`), and `test-integration` (`go test -tags integration ./...`) targets
- [x] 1.3 Verify `make test` passes on the empty module

## 2. Provider contract (`internal/provider`)

- [x] 2.1 Define `Request` (Prompt, Material — by value, per design D2)
- [x] 2.2 Define `Provenance` (Provider, Model, PromptHash) and `Result` (Findings, Provenance) per D3
- [x] 2.3 Define the `Provider` interface (`Review(ctx, Request) (*Result, error)`) per D1
- [x] 2.4 Define sentinel errors (`ErrUnavailable`, `ErrAuth`) and document the ran/did-not-run semantics per D4
- [x] 2.5 Implement `HashPrompt` helper with unit tests

## 3. Conformance suite (`internal/provider/providertest`)

- [x] 3.1 Implement `Conform(t, newProvider)` skeleton with contract checks: provenance populated, empty findings yield result-not-error, did-not-run yields error-not-result, cancellation returns promptly
- [x] 3.2 Implement the cold-reviewer canary checks per D6: instruction-file canary in the suite's working directory, repo-file canary referenced by but not included in the material
- [x] 3.3 Gate real-reviewer execution behind the `integration` build tag; keep the default run hermetic

## 4. Reference provider and suite validation

- [x] 4.1 Implement the `loopback` reference provider (in-process, honest by default) per D7
- [x] 4.2 Add rig options for each violation: leak instruction file, read referenced repo path, omit provenance, ignore cancellation
- [x] 4.3 Suite-of-the-suite tests: `Conform` passes honest loopback, fails each rigged variant

## 5. Wrap-up

- [x] 5.1 Run `make test` and `gofmt -l .` clean
- [x] 5.2 Update `AGENTS.md` milestones (check off "Provider interface and conformance suite"; replace the pre-code architecture note with the real `internal/` layout)
