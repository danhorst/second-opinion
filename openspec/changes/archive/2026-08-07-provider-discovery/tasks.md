## 1. Usage text

- [x] 1.1 Convert the `usage` constant in `cmd/second-opinion/main.go` into a composed string or function that renders `review.Providers()` on the `--provider` line.
- [x] 1.2 Add the `--list-providers` line to the usage block.
- [x] 1.3 Update existing tests that assert usage content to assert it names each registered provider, rather than matching fixed text.

## 2. The flag

- [x] 2.1 Add `--list-providers` to `parseArgs`, as a config field alongside `help`.
- [x] 2.2 In `run`, handle it immediately after the help check — before the target check, provider selection, prompt loading, and material assembly.
- [x] 2.3 Print one provider name per line to stdout with no header or decoration, and return 0.

## 3. Tests

- [x] 3.1 Unit-test `parseArgs` accepts `--list-providers` with no target and no other flags.
- [x] 3.2 Test `run` with only `--list-providers`: stdout is the sorted registry one per line, stderr is empty, exit 0.
- [x] 3.3 Test it succeeds with `SECOND_OPINION_PROVIDER` unset and with it set to an unregistered name — neither affects the output.
- [x] 3.4 Test the round trip: every line of the output is accepted by `--provider` without an unknown-provider error.
- [x] 3.5 Test the usage block names every registered provider.

## 4. Skill

- [x] 4.1 Fix `skills/second-opinion/SKILL.md` — replace "Run without a provider to see what is registered" with `second-opinion --list-providers`.
- [x] 4.2 Confirm the wording stays provider-neutral: the skill presents the list as a set the agent chooses from, never one as a default.
- [x] 4.3 Confirm the embedded-copy test still passes, since the repository and embedded skill must stay byte-identical.

## 5. Verify

- [x] 5.1 `make test` passes and `gofmt` is clean.
- [x] 5.2 Run `second-opinion --list-providers` from a directory outside any git repository and confirm exit 0 with the three names.
