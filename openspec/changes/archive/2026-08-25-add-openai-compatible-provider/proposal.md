## Why

The current provider roster is subprocess-oriented, which makes strict material-only review impossible without granting a model access to a local filesystem.
An HTTP provider can send the assembled request by value and receive findings without starting a reviewer process in the caller's environment.

## What Changes

- Add a generic OpenAI-compatible HTTP provider with configurable base URL, API key, and model.
- Support the common non-streaming chat-completion request/response subset shared by OpenAI-compatible gateways.
- Use OpenRouter as a supported endpoint configuration without hard-coding OpenRouter-specific transport logic.
- Report the response's actual model in provenance and preserve the shared prompt and material hashes.
- Classify authentication, rate-limit, timeout, unavailable, and malformed-response failures consistently with the provider contract.
- Register the provider with the CLI and document its environment variables and endpoint configuration.
- Add hermetic HTTP tests using a local test server and integration coverage for opt-in authenticated endpoints.

## Non-goals

- Adding strict-isolation policy or requiring callers to select a strict provider.
- Implementing streaming, tool calls, file uploads, vision, or provider-specific extensions in the first version.
- Adding a dedicated OpenRouter adapter.
- Selecting models, routing requests, or ranking providers automatically.
- Changing the findings format or implementing MCP.

## Capabilities

### New Capabilities

- `openai-compatible-provider`: Review assembled material through a configurable OpenAI-compatible HTTP endpoint.

### Modified Capabilities

None.

## Impact

- New provider implementation under `internal/provider` and registry wiring under `internal/review`.
- CLI configuration and README documentation for endpoint and API-key environment variables.
- No third-party SDK is required; the implementation can use Go's standard HTTP client.
- Review material will leave the machine when configured with a remote endpoint, consistent with the existing Claude, Codex, and Antigravity providers.
