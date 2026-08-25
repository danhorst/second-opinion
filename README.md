# second-opinion

Get an adversarial second opinion on a document or a diff from a model that did not write it.
The premise is model-level: a model cannot reliably review its own work, so the reviewer is always a different model than the author.
The tool has no home model — Claude is a provider like any other.

Three rules shape the design:

- The reviewer runs cold.
  Everything it sees is explicitly assembled into the material, with provider-specific controls suppressing implicit project files, configuration, and session history.
  Strict prevention of deliberate filesystem reads is a separate capability and is not claimed by the current Codex adapter.
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

To run the optional live OpenAI-compatible integration test, put the API settings in an untracked `.env` file and run `make test-integration-openai`.
The target loads `.env` only for that command and skips the live test when the required key or model is absent.

## Usage

```
usage: second-opinion [flags] PATH...
       second-opinion [flags] --diff [BASE]

  PATH...        files to review, concatenated in argument order
  --diff [BASE]  review the git diff against BASE (default: HEAD)
  --provider P   reviewer provider; defaults to $SECOND_OPINION_PROVIDER
  --model M      force a model (provider semantics apply)
  --prompt-file  replace the baked review prompt with a file's contents

subcommands:
  skill install  install the calling-agent skill for claude/codex harnesses

exit codes: 0 review completed; 1 reviewer did not run; 2 usage error
```

Findings go to stdout; one-line provenance naming the provider and model that actually ran goes to stderr.
Forcing a model with `--model` is a foot-gun: providers can silently reject models depending on account auth, and the tool falls back to the provider default only for the one failure it can detect.

## Providers

- `claude` — shells out to `claude -p` with tools and setting sources disabled, so the reviewer cannot read the repo or inherit user memory.
- `antigravity` — shells out to `agy --print` from an empty temp directory.
- `codex` — shells out to `codex exec` from an empty temporary directory with project-doc loading, user config, and execution-policy rules suppressed, plus a read-only ephemeral sandbox.
  Current Codex JSONL output does not identify the selected default model, so unforced reviews report the explicit `codex-default` marker.
- `openai-compatible` — sends prompt and material to a configurable non-streaming chat-completions endpoint without local reviewer filesystem access.
  Configure `SECOND_OPINION_API_KEY`, optional `SECOND_OPINION_API_BASE_URL`, and `SECOND_OPINION_API_MODEL`.
  The default endpoint is OpenAI; OpenRouter works with `SECOND_OPINION_API_BASE_URL=https://openrouter.ai/api/v1`.

`make test-integration-openai` is the explicit credential-bearing validation path; ordinary tests use a local HTTP fixture and never require API credentials.

Select one with `--provider` or `$SECOND_OPINION_PROVIDER`.
The API provider is opt-in and sends review material to the configured remote endpoint.

## Calling-agent skill

`second-opinion skill install [--harness claude|codex|all]` installs the triage skill that teaches a calling agent the discipline: the external model finds, the caller filters.
Antigravity consumes the same skill as a plugin via `agy plugin install`.

## License

[MIT-0](LICENSE)

[1]: https://brew.sh
[2]: https://github.com/danhorst/homebrew-tap
[3]: https://go.dev
