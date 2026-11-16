## 1. Provider implementation

- [x] 1.1 Add the `openai-compatible` provider with base URL, API key, model, and HTTP client options.
- [x] 1.2 Resolve `SECOND_OPINION_API_BASE_URL`, `SECOND_OPINION_API_KEY`, and `SECOND_OPINION_API_MODEL` with CLI model precedence.
- [x] 1.3 Build the non-streaming chat-completion request with prompt/material text and no tools or file inputs.
- [x] 1.4 Decode findings and response-model provenance, including the explicit unknown-model marker.
- [x] 1.5 Classify transport, HTTP status, cancellation, and malformed-response failures without leaking credentials.

## 2. Tests

- [x] 2.1 Add local HTTP-server tests for configuration, URL, authorization, request body, and no-tool behavior.
- [x] 2.2 Add response parsing tests for findings, actual model, missing model, empty choices, and malformed JSON.
- [x] 2.3 Add failure tests for authentication, rate limits, timeouts, server errors, bounded diagnostics, and cancellation.
- [x] 2.4 Add the integration-tagged conformance test for a configured authenticated endpoint.

## 3. CLI and documentation

- [x] 3.1 Register `openai-compatible` and update provider-list and model configuration tests.
- [x] 3.2 Document endpoint, API-key, model, locality, and OpenRouter configuration in README and project architecture notes.
- [x] 3.3 Add the provider to the supported-provider OpenSpec context without making it the default.

## 4. Validation

- [x] 4.1 Run `gofmt`, `go vet ./...`, and the hermetic test suite.
- [x] 4.2 Add `make test-integration-openai`, which loads an opt-in `.env` and runs the live endpoint test when credentials are available.
- [x] 4.3 Run strict OpenSpec validation.
