## Context

The existing adapters launch local model harnesses, so even an empty working directory does not constitute a strict filesystem boundary.
An HTTP provider can receive the prompt and material as request content without a reviewer subprocess in the caller's environment.

OpenRouter and OpenAI expose compatible chat-completion interfaces, but gateway-specific routing and metadata must not leak into the provider contract.

## Goals / Non-Goals

**Goals:**

- Add one configurable HTTP adapter usable with OpenAI, OpenRouter, and compatible gateways.
- Send only the rendered prompt and material as text content, with no tools or file references.
- Preserve provider, actual model, prompt hash, and material hash provenance.
- Keep the first implementation standard-library-only and non-streaming.
- Make failures classifiable and testable without mocking the provider boundary.

**Non-Goals:**

- Strict-isolation policy enforcement; that is the next change.
- Streaming, tools, files, multimodal inputs, or provider-specific request extensions.
- Automatic model discovery, routing, retries, or provider selection.

## Decisions

### D1 — Use a generic OpenAI-compatible adapter

Name the provider `openai-compatible` and configure the endpoint with a base URL.
OpenRouter is therefore a configuration, not a separate implementation.

Alternative: add an OpenRouter adapter first.
Rejected because it would duplicate the wire protocol and make strict isolation depend on a vendor-specific name.

### D2 — Use the chat-completions subset over `net/http`

Send a non-streaming `POST <baseURL>/chat/completions` request with two messages: the review prompt as a system message and the material as a user message.
Do not send tools, file paths, or provider extensions.
Decode the first choice's text content and the response model.

Alternative: use the OpenAI SDK or implement Responses first.
Rejected for v1 because the standard library keeps the dependency surface small and chat completions are the broadest compatible subset.

### D3 — Configure through explicit environment variables

Use `SECOND_OPINION_API_BASE_URL` with `https://api.openai.com/v1` as the default, `SECOND_OPINION_API_KEY` for the bearer token, and `SECOND_OPINION_API_MODEL` when `--model` is not supplied.
The CLI model argument takes precedence over the environment model.
Missing key, endpoint, or model is a usage/configuration failure before any request is sent.

Alternative: infer provider-specific environment names such as `OPENROUTER_API_KEY`.
Rejected because it couples the generic adapter to one gateway and makes secret selection ambiguous.

### D4 — Classify HTTP failures at the adapter boundary

Map network errors, timeouts, 408, 429, and 5xx responses to `ErrUnavailable`; map 401 and 403 to `ErrAuth`; return other non-success statuses and malformed successful responses as detailed provider errors.
The response body is bounded before inclusion in diagnostics.

### D5 — Preserve actual response provenance

Use the response's `model` field when present.
If a compatible endpoint omits it, report the explicit `openai-compatible-model-unknown` marker rather than claiming the requested model actually ran.

### D6 — Test transport locally and endpoint behavior remotely

Use `httptest.Server` for request-shape, response parsing, cancellation, and error classification tests.
Add an integration-tagged test that runs the shared conformance suite against the configured authenticated endpoint when credentials are available.

## Risks / Trade-offs

- [OpenAI-compatible implementations diverge] → Limit v1 to the common non-streaming chat-completions subset and fail with a clear protocol error when required fields are absent.
- [Remote endpoints receive sensitive material] → Document provider locality and require explicit provider selection; strict-isolation policy remains a separate caller-level decision.
- [API key leaks through diagnostics] → Never include request headers or environment values in errors; cap response-body diagnostics.
- [Router substitutes a model] → Prefer the response model and use an explicit unknown marker when absent.
- [Remote latency and rate limits] → Honor request context and classify transient statuses as unavailable; do not add implicit retries in v1.

## Migration Plan

No data migration is required.

Add the provider to the registry without changing the default provider behavior.
Users opt in with `--provider openai-compatible` and configure credentials and model selection through the documented environment variables.

## Open Questions

The strict-isolation change must decide whether this provider's lack of local filesystem access is sufficient for a `strict` capability and how to represent network/data-residency constraints separately.
