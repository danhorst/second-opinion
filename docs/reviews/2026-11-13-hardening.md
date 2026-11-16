# Provider hardening follow-up — 2026-11-13

The follow-up prioritizes false success, cancellation, and cold-context evidence before adding MCP or another provider. Three implementation tasks were delegated to smaller-model agents; all encountered a usage limit. Two left proposal scaffolding, which was completed by the primary agent along with implementation and validation. Existing work was preserved.

## Implemented

- Codex accepts recognized message events only and requires successful terminal completion. Malformed JSON, scanner overflow, missing messages, incomplete turns, and explicit failures return errors; an explicit empty message remains a valid review. Nonfatal item warnings remain compatible.
- Claude requires a successful result envelope with explicit `is_error=false` and a present string result. Raw result whitespace survives. Expired OAuth maps to `ErrAuth`.
- All subprocess adapters use shared process handling: a new process group on macOS/Linux/BSD, group termination on cancellation and completion, and a two-second inherited-pipe drain limit. Other platforms retain parent cancellation plus bounded pipe draining. Argument-limit errors preserve their cause and explain how to reduce input.
- Shared conformance now checks absolute file paths as well as relative references, with the same positive-control gate. Nil results fail instead of crashing the suite. Tests demonstrate that deliberate reference-provider leaks are caught.
- Codex ignores user configuration/rules, suppresses host skill discovery, and disables optional local tools, memory, apps, plugins, subagents, skill search, and web search. Authentication is retained; configured user model defaults are intentionally ignored with user config.
- Claude uses safe mode, disabled skills, strict MCP configuration without supplied servers, and disabled session persistence alongside existing empty tools/settings. Its composed request is supplied on stdin without a positional prompt, removing the material's OS argument limit.
- Codex skill installation now writes `~/.agents/skills/second-opinion/SKILL.md` with frontmatter intact. Existing legacy custom prompt files remain untouched. Installer tests use temporary homes; no real user-home installation was performed.

The native skill root follows [official skill documentation](https://learn.chatgpt.com/docs/build-skills). The [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference) documents shell/image/web controls; additional flags were checked against installed CLI help and feature listings. Tested versions: codex-cli 0.167.0-alpha.4.1 and Claude Code 2.1.234. Older binaries lacking these controls fail closed.

## Real-provider evidence

Each suite was run independently with a 120-second test deadline and synthetic material/canaries only:

| Provider | Outcome |
| --- | --- |
| Codex | Passed contract, pre-cancellation, positive control, instruction-file, relative-path and absolute-path cold checks; 33.75 seconds |
| Antigravity | Passed the same stronger suite; 54.51 seconds |
| Claude | Failed contract and positive control because OAuth expired and could not refresh. All cold checks were correctly inconclusive. No claim of successful stdin delivery or isolation certification is made for this attempt. |

Reauthenticate Claude using its normal login flow, then rerun `go test -tags integration ./internal/provider/claude -run '^TestClaudeConformance$' -timeout 120s -v`. The implementation does not automatically weaken flags or switch authentication to work around this failure.

## Local validation

- `make test` (vet and default tests) passed.
- `go test -race ./...` passed, including actual subprocess-tree tests with a readiness handshake.
- Integration-tagged packages compiled without executing tests.
- Windows and Linux cross-builds passed; these prove compilation, not cross-platform runtime cleanup.
- All three adapter entry points return `ErrUnavailable` with no result when their executables are genuinely absent from PATH; no fake binaries are used.
- OpenSpec strict validation passed; provider-result-validation, provider-process-control, and cold-review-boundaries were synced to current specs.

Build caches were redirected to `/private/tmp` where needed for sandbox access. No provider executable was mocked. Process tests use real test-executable subprocesses solely to verify OS lifecycle behavior. Hosted CI was not run from this workspace.

## Remaining work and scope decisions

1. **Global-context evidence:** the stronger canary pass is observational, not exhaustive. Antigravity still lacks explicit global-context/tool suppression comparable to the other adapters. Isolated-home/global-config canaries need careful authentication handling; no real user configuration was modified to simulate them.
2. **Cancellation limits:** descendants that daemonize into another process group can escape cleanup. Windows needs Job Objects for tree termination. The Provider API exposes no readiness signal, so shared pre-cancellation checks do not prove in-flight model cancellation; actual process readiness is tested in the lifecycle helper instead.
3. **Large input/output:** Claude no longer puts material in argv. Antigravity still does; argument-limit failures are actionable, but streaming transport and material/output budgets need follow-up. No truncation or automatic review splitting was introduced.
4. **Provenance:** Codex/Antigravity requested model names and default markers are not independently verified model identities. Antigravity now advertises JSON output; inspect its real envelope before claiming stronger provenance.
5. **MCP/Ollama:** deferred as distinct features. MCP can lead with raw text and provenance without waiting for heuristic findings parsing. Ollama needs explicit local endpoint/model selection and real-endpoint conformance before being called complete. The existing findings-envelope proposal remains untouched.

No changes were committed or released. The completed OpenSpec changes remain available for inspection.
