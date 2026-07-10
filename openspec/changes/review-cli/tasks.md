# review-cli — tasks

## 1. Engine pieces (`internal/review`)

- [ ] 1.1 Baked prompt constant with transport-neutral wording per design D5; unit test that no transport phrase appears
- [ ] 1.2 `FromFiles` (headers, argument order) and `FromDiff` (git diff BASE) per D4, with empty-material refusal; unit tests including a temp git repo for diff
- [ ] 1.3 Provider registry (`Providers`, `NewProvider`) per D2; unit tests for unknown names and model passthrough

## 2. CLI (`cmd/second-opinion`)

- [ ] 2.1 Flag parsing and selection per D3 (flag > env > refuse-with-list); usage errors exit 2
- [ ] 2.2 Wiring: assemble material, build request, run review, findings to stdout, provenance line to stderr, exit codes per D6
- [ ] 2.3 Wiring test against the Loopback reference provider per D7

## 3. Verification

- [ ] 3.1 Integration-tagged smoke test: one real review end-to-end through one provider
- [ ] 3.2 Manual run: `go run ./cmd/second-opinion --provider claude --diff` on this repo (or a doc review), observed working

## 4. Wrap-up

- [ ] 4.1 `make test` and `gofmt -l .` clean
- [ ] 4.2 Update `AGENTS.md` (Architecture, milestone); note the dotfiles retirement as a manual follow-up outside this repo
