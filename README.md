# second-opinion

Get an adversarial second opinion on a document or a diff from a model that did not write it.
The premise is model-level: choose a reviewer model different from the author. The tool reports reviewer provenance but does not enforce that choice.
The tool has no home model — Claude is a provider like any other.

Three rules shape the design:

- The reviewer runs cold.
  Everything it sees was explicitly assembled into the material — no auto-loaded project files, no repo access, no session history.
- The reviewer finds; it does not rank.
  Severity is the reviewer's guess, and novelty is the axis it cannot judge, because it cannot see what the author already knew.
- Failure is loud.
  Exit status distinguishes "the reviewer ran and found nothing" from "the reviewer did not run."

## Installation

There is a [Homebrew][1] formula for `second-opinion` in [`danhorst/homebrew-tap`][2]:

```bash
brew tap danhorst/tap && brew install danhorst/tap/second-opinion
```

Without Homebrew, build from source with a recent [Go][3] toolchain: `go build ./cmd/second-opinion`.

## Usage

```
usage: second-opinion [flags] PATH...
       second-opinion [flags] --diff [BASE]

  PATH...        files to review, concatenated in argument order
  --diff [BASE]  review the git diff against BASE (default: HEAD)
  --provider P   reviewer provider; defaults to $SECOND_OPINION_PROVIDER
  --model M      force a model (provider semantics apply)
  --prompt-file F replace the baked review prompt with a file's contents
  --json         emit raw findings and full provenance as JSON
  --dry-run      preview the assembled request as JSON; no reviewer runs
  --timeout D    limit the provider call (e.g. 2m); default: unlimited
  --             treat remaining arguments as literal file paths
  --version      print application version and available Git build identity
  --list-providers print registered provider names and exit

subcommands:
  mcp init       initialize the private HTTP service token
  mcp serve      serve stateless MCP tools (stdio or private HTTP)
  skill install  install the calling-agent skill for claude/codex harnesses

exit codes: 0 success (review, preview, or discovery); 1 review/output failure;
            2 usage error
```

Findings go to stdout; one-line provenance naming the provider and model that actually ran goes to stderr.
Forcing a model with `--model` is a foot-gun: providers can silently reject models depending on account auth, and the tool falls back to the provider default only for the one failure it can detect.

### Coding harnesses

```bash
second-opinion --dry-run --diff HEAD
second-opinion --provider claude --json --timeout 2m --diff HEAD
second-opinion --provider codex --json -- spec.md supporting-file.go
```

`--json` emits one object on successful review:

```json
{"schema_version":1,"kind":"review","raw":"reviewer text","isolation":"unverified","provenance":{"provider":"claude","model":"reported-model-id","prompt_hash":"full SHA-256 hash"}}
```

`raw` is the provider-returned text, not a parsed finding list. Provenance also includes the full SHA-256 `material_hash`. Read the exit status first: failures have a stderr diagnostic and no success object. The usual stderr provenance line is also emitted in JSON mode. Help and plain provider discovery retain their text output; `--list-providers --json` includes the `provider_isolation` mapping. JSON output requires UTF-8 text and fails rather than silently replacing invalid bytes.

`--dry-run` emits `schema_version`, `kind: "request"`, `prompt`, `material`, and `prompt_hash`. It requires no provider or credentials and ignores provider/model selection. This is the assembled engine request; adapter delimiters and harness-internal instructions are not included. Preview and execution assemble separate snapshots, so keep inputs unchanged between them.

`--timeout` takes a positive Go duration such as `90s` or `2m`. Its budget covers the provider call and fallback attempts, starting after target assembly. On macOS, Linux, and BSD, cancellation terminates the invocation process group; inherited output-pipe waits are bounded to two seconds. Detached children outside that group are not covered. Other platforms currently cancel only the parent process.

Diff mode includes tracked staged and unstaged changes against the base, **not untracked files**. Use file mode for new files and deliberately selected supporting context. Git color, external diff, and text conversion are disabled. Use `--` for filenames beginning with a dash; for a prompt-file value use a path such as `./-prompt.txt`.

Application version is shared by `--version` and MCP discovery. Git checkout builds include available revision and dirty-state metadata. `scripts/release vX.Y.Z` stamps the selected version into source archives; protocol and JSON schema versions remain separate.

### Harness setup

Register the installed binary with a supported harness, then check its connection:

```sh
second-opinion mcp install --harness codex --dry-run
second-opinion mcp install --harness codex
second-opinion mcp doctor --harness codex
```

Supported values are `codex`, `claude` (Claude Code), and `antigravity` (CLI/IDE).
The default is user scope and stdio: no service or token is needed. Use `--scope project`
from the project root for a project registration. Installation never selects a reviewer,
changes tool trust, or installs the calling-agent skill; `skill install` remains separate.
Restart/reload the harness and approve its normal MCP trust prompt as needed.

| Harness | User configuration | Project configuration |
| --- | --- | --- |
| Codex | `$CODEX_HOME/config.toml`, otherwise `~/.codex/config.toml` | `.codex/config.toml` |
| Claude Code | `~/.claude.json` | `.mcp.json` |
| Antigravity CLI/IDE | `~/.gemini/config/mcp_config.json` | `.agents/mcp_config.json` |

Use `--config PATH` for a different configuration file or Antigravity surface. A custom
`CLAUDE_CONFIG_DIR` requires an explicit `--config` path. These adapters target the documented
formats ([Codex](https://developers.openai.com/codex/mcp/),
[Claude Code](https://code.claude.com/docs/en/mcp),
[Antigravity](https://antigravity.google/docs/mcp)); they do not certify that every installed
harness version accepts the configuration.

For the shared Homebrew HTTP service:

```sh
second-opinion mcp init
brew services start danhorst/tap/second-opinion
second-opinion mcp install --harness codex --transport http
second-opinion mcp doctor --harness codex --json
```

HTTP registrations launch `second-opinion mcp connect`, a stdio bridge that reads the private
token file and connects to `http://127.0.0.1:8765/mcp`. The harness configuration contains the
file path, never its contents. GUI applications need no exported token variable. Supply
`--url` and `--token-file` when installing for a custom service address/token location.
The connector accepts only literal loopback HTTP endpoints and refuses redirects and proxies.
`mcp init` and service startup remain explicit operations.

All registration commands support `--json`; install and uninstall also support `--dry-run`.
Repeated installation is a no-op. A conflicting entry requires `--replace`. Existing files
are backed up privately beside the original before atomic replacement; JSON/TOML formatting
(and TOML comments) can change, but unrelated setting values are preserved. Malformed files,
duplicate JSON keys, and symlink configuration files are refused. Avoid editing the same
configuration concurrently; the installer checks for changes before publication but does
not lock out another application. Project entries contain machine-specific absolute paths;
review these before sharing them. Homebrew executable paths use the stable `opt` location.

`mcp doctor` checks the selected file, executable, token safety when applicable, initialization,
and discovery of `review` and `list_providers`. It runs no review and verifies neither provider
authentication nor live harness acceptance. Other configuration scopes or managed policy can
still override this registration. Diagnostics return nonzero on failure.

```sh
second-opinion mcp uninstall --harness codex
```

Uninstall removes only a recognized entry using this executable, leaving other servers,
the token, and the service intact. If the executable moved, supply its old absolute path
with `--executable`. Cursor and VS Code installers are deferred; use manual MCP configuration.

### Stateless MCP

Start a local stdio server with `second-opinion mcp serve`. A harness configuration that supports the common MCP server format can use:

```json
{"mcpServers":{"second-opinion":{"command":"second-opinion","args":["mcp","serve"]}}}
```

The server supports MCP `2026-07-28` and `2025-11-25`. Discover `list_providers` with `{}`; it returns sorted registered names, a `provider_isolation` mapping, and effective limits without checking credentials or invoking a model. Then call `review` with complete material:

```json
{"provider":"codex","material":"Complete document or patch, with deliberately selected supporting context","timeout_ms":120000}
```

Optional `model` selects a reviewer; optional `prompt` replaces the baked prompt. Provider is always explicit, even when `SECOND_OPINION_PROVIDER` is set. The gateway never opens paths or URLs in material, requests client roots, or inherits previous calls. Assemble diffs and supporting files in the calling harness.

Each result has equivalent `structuredContent` and one JSON text block. Success contains `schema_version: 1`, `kind: "review"`, authoritative `raw` text, effective `isolation`, unchanged `provenance`, and a `request` receipt containing `material_hash`, `material_bytes`, and effective `timeout_ms`. A successful empty `raw` means a validated review completed. Model provenance has the same limitations described below; unknown default markers do not establish independence.

Tool failures set `isError` and return `kind: "error"` with `error.code`, a safe message, and `execution`. Input errors and `busy` mean `not_started`; provider errors, timeouts, cancellation, and output overflow mean `unknown`. A lost response also means unknown completion. There are no result handles, recovery, cached results, or idempotency keys. Retrying starts independent work and may incur duplicate cost. The gateway adds no retries; the existing Codex model-auth fallback remains within the call deadline.

Limits: 8 MiB request frames/bodies; 1 MiB combined material and resolved prompt (UTF-8 bytes); 256-byte model names; 120-second default/300-second maximum timeout; two active reviews and four admitted transport messages; 8 MiB stdout/64 KiB stderr captured per provider invocation; 1 MiB raw findings; 16 MiB encoded responses. Excess input/output fails without truncation. A third review returns `busy`, with no waiting queue. Disconnect or server shutdown cancels active work; shutdown waits at most five seconds. Provider process-cleanup limitations still apply.

For private HTTP, generate a random bearer token outside argv and configure the harness to send it on every request:

```bash
export SECOND_OPINION_MCP_TOKEN="$(openssl rand -hex 32)"
second-opinion mcp serve --transport http --listen 127.0.0.1:8765
```

Use endpoint `http://127.0.0.1:8765/mcp` and header `Authorization: Bearer <the token>` in the harness's HTTP transport configuration, preferably through its secret/environment facility. Do not commit the token. HTTP requires at least 32 token bytes, literal `127.0.0.1` or `::1` binding, exact Host authority, and no Origin header. It is single-operator loopback serving, not public OAuth hosting. The gateway strips its token from provider child environments. HTTP is stateless with no session persistence or GET event stream; callers cancel by closing the originating request. Header/body reads are bounded to ten seconds, writes to 310 seconds, and idle connections to 30 seconds.

### Homebrew background service

Starting with v0.2.1, the formula includes a per-user service launcher and `mcp init` token setup. Homebrew's [service block](https://docs.brew.sh/Formula-Cookbook#service-files) handles launchd on macOS and user systemd on Linux.

Initialize the service credential, then start the service:

```bash
second-opinion mcp init
brew services start danhorst/tap/second-opinion
```

`mcp init` creates `~/.config/second-opinion/mcp-token` from 32 cryptographically random bytes, with mode 0600 and private newly created directories. Re-running it validates and preserves an existing token. It never prints the secret, starts a service, or edits harness settings. Insecure or malformed existing files are left untouched with an error. The command supports macOS and Linux, matching the service launcher.

Use `second-opinion mcp init --token-file /absolute/private/path` for a custom location; otherwise `SECOND_OPINION_MCP_TOKEN_FILE` overrides the default. Configure the service manager with that same custom path. There is no implicit token rotation: deliberately replace a token only when ready to update every client and restart the service.

Run as your normal user, without `sudo`. Prefer `mcp install --harness codex --transport http` (or another supported harness) to avoid copying the token. For manual HTTP configuration, configure the harness for `http://127.0.0.1:8765/mcp` with `Authorization: Bearer <contents of token file>`, preferably using its secret facility. The token is deployment configuration; the gateway remains stateless. The launcher accepts a 32–4096-character bearer token in a user-owned regular file with mode 0400 or 0600, with at most one trailing newline. It rejects symlinks and never copies the token into the generated plist, service argv, or logs. Existing provider-child token stripping still applies.

```bash
brew services info danhorst/tap/second-opinion
brew services restart danhorst/tap/second-opinion
brew services stop danhorst/tap/second-opinion
```

The service starts at user login. Automatic crash restart is disabled so a missing token does not cause a restart loop; fix configuration and restart explicitly. Startup diagnostics go to `$(brew --prefix)/var/log/second-opinion.log`. Token changes take effect on restart; update the harness token at the same time.

Provider binaries must be available in the Homebrew/system PATH or `~/.local/bin`, with credentials already configured for that user. Interactive shell profiles and PATH changes are not loaded. For tools installed elsewhere, create a deliberate symlink in `~/.local/bin` or configure the service environment. The launcher supports `SECOND_OPINION_MCP_TOKEN_FILE` and `SECOND_OPINION_MCP_LISTEN` overrides supplied by the service manager; simply exporting them in a terminal does not configure launchd. Listener overrides remain subject to the server's literal-loopback restriction.

To diagnose startup in the foreground after installation:

```bash
"$(brew --prefix second-opinion)/libexec/second-opinion-mcp-service"
```

The launcher uses `exec`, so Homebrew stop signals reach the MCP server and its existing bounded shutdown. Starting this background HTTP service does not replace or affect harness-owned stdio servers.

## Providers

- `openai-compatible` — configurable non-streaming HTTP chat-completions endpoint, including OpenRouter. Set `SECOND_OPINION_API_BASE_URL`, `SECOND_OPINION_API_KEY`, and `SECOND_OPINION_API_MODEL` (or select `--model`). Endpoint credentials and real endpoint conformance are deployment-specific.

- `claude` — sends the composed request on stdin to `claude -p` with safe mode, tools, skills, and settings disabled, strict MCP configuration, and session persistence disabled. Requires support for these controls (inspected in 2.1.234); the current real-binary retest is blocked by expired OAuth.
- `antigravity` — shells out to `agy --print` from an empty temp directory.
- `codex` — uses an isolated cwd, ignores user config/rules, suppresses host skills and optional tools/context, and keeps a read-only ephemeral sandbox. Verified with codex-cli 0.167.0-alpha.4.1; stronger conformance including absolute-path isolation passed 2026-11-13. Older binaries missing these controls fail closed.

Antigravity also passed the absolute-path conformance check on 2026-11-13; that observation does not prove suppression of every harness-global context source. Claude and Codex reject malformed or incomplete transport output instead of treating it as a clean empty review.

Select one with `--provider` or `$SECOND_OPINION_PROVIDER`.

`codex-default` and `antigravity-default` are unknown-model markers, not model identities. Codex prefers reported session model metadata and falls back to the requested name or its default marker. Antigravity forced names reflect the requested model without independently verified output metadata. Neither establishes that the reviewer differs from the author. Claude uses model usage metadata when available and falls back to a requested name or `claude-default` if it is absent.

Ollama and the optional parsed findings envelope remain planned. See the [repository review](docs/reviews/2026-11-13.md) and its [hardening follow-up](docs/reviews/2026-11-13-hardening.md) for remaining gaps and verification limits.

## Local material isolation

To require a client boundary after assembling the material:

```bash
second-opinion --provider openai-compatible --require-isolation local-material-only --json document.md
second-opinion --list-providers --json
```

For MCP, add `"require_isolation": "local-material-only"` to the `review` arguments. Unsupported providers fail before reviewer execution: CLI exit 1; MCP `isolation_unsupported` with `execution: "not_started"`. There is no downgrade or provider fallback. Omit the requirement to retain existing provider selection.

| Provider | Reported local boundary |
| --- | --- |
| `openai-compatible` | `local-material-only` |
| `codex`, `claude`, `antigravity` | `unverified` |

`local-material-only` means the adapter sends only the assembled prompt and material as review content. It launches no reviewer harness, supplies no tools, and never follows reviewer-provided paths or URLs. Redirect responses and tool/function-call responses fail explicitly, including tool calls accompanied by findings. Configure the final endpoint URL directly.

This is a **client-side boundary**. The configured endpoint, credentials, configuration, and HTTP transport are trusted. A local endpoint may have its own filesystem access; a remote endpoint may add context or use server-side tools. This client does not certify either behavior, protect against compromised local code, or keep remote reviews on the machine. Local transport tests are not remote endpoint or model certification.

`unverified` means the adapter has no certified local read boundary; its existing cold-invocation controls remain required. Successful CLI/MCP JSON includes `isolation`, and CLI stderr reports it alongside provenance. `--dry-run` records a supplied `require_isolation` but does not check support or claim execution. Review output and discovery describe capability, not credential readiness.

See [verification evidence and limits](docs/reviews/2026-11-17-material-isolation.md).

## Calling-agent skill

`second-opinion skill install [--harness claude|codex|all]` installs the triage skill that teaches a calling agent the discipline: the external model finds, the caller filters.
Codex receives a native skill at `~/.agents/skills/second-opinion/SKILL.md`, with discovery frontmatter intact. Existing legacy `~/.codex/prompts/second-opinion.md` files are preserved. Antigravity consumes the same skill as a plugin via `agy plugin install`.

## License

[MIT-0](LICENSE)

[1]: https://brew.sh
[2]: https://github.com/danhorst/homebrew-tap
[3]: https://go.dev

## Releasing

Merge and validate changes on `main` before tagging. See [release instructions](docs/releasing.md); the script and publishing workflow both enforce main ancestry.
