# openai-compatible-provider

## ADDED Requirements

### Requirement: Configurable OpenAI-compatible endpoint
The provider SHALL send reviews to a configurable base URL, defaulting to `https://api.openai.com/v1`, using a bearer token from `SECOND_OPINION_API_KEY`.
The provider SHALL use `SECOND_OPINION_API_MODEL` when no model is supplied by the caller.

#### Scenario: OpenRouter endpoint configuration
- **WHEN** the base URL is `https://openrouter.ai/api/v1` and an API key and model are configured
- **THEN** the provider sends the review to that endpoint with the configured bearer token and model

#### Scenario: Missing configuration fails before transport
- **WHEN** the API key or model is missing
- **THEN** the provider returns a configuration error without making an HTTP request

### Requirement: Review material travels as text content
The provider SHALL send the prompt and material as text message content in a non-streaming chat-completion request.
The request MUST NOT include tools, file paths, repository references, or uploaded files.

#### Scenario: Request contains only assembled review inputs
- **WHEN** a review is requested
- **THEN** the HTTP request contains the rendered prompt and exact material content, with streaming disabled and no tool declarations

### Requirement: Successful response returns findings and provenance
The provider SHALL return the first assistant text choice as findings and SHALL populate provider, prompt hash, and material hash provenance.
It SHALL use the response's model field for model provenance and SHALL use `openai-compatible-model-unknown` when that field is absent.

#### Scenario: Successful completion
- **WHEN** the endpoint returns a valid completion with assistant text and a model
- **THEN** the provider returns that text and reports the response model and both request hashes

#### Scenario: Endpoint omits model identity
- **WHEN** the endpoint returns a valid completion without a model field
- **THEN** the provider returns the findings with `openai-compatible-model-unknown` as the model marker

### Requirement: HTTP failures are classified
The provider SHALL classify network errors, timeouts, 408, 429, and 5xx responses as `ErrUnavailable` and 401 or 403 responses as `ErrAuth`.
It SHALL preserve bounded diagnostic detail without exposing credentials.

#### Scenario: Authentication rejection
- **WHEN** the endpoint returns 401 or 403
- **THEN** the provider returns an error matching `ErrAuth` without a result

#### Scenario: Rate limiting
- **WHEN** the endpoint returns 429
- **THEN** the provider returns an error matching `ErrUnavailable` without an implicit retry

#### Scenario: Malformed successful response
- **WHEN** the endpoint returns 2xx without a usable assistant text choice
- **THEN** the provider returns a protocol error and no result

### Requirement: HTTP requests are cancellable
The provider SHALL bind each HTTP request to the caller's context and SHALL return promptly when that context is cancelled.

#### Scenario: Caller cancels an in-flight request
- **WHEN** the caller cancels the review context while the endpoint is waiting
- **THEN** the provider returns the cancellation error without waiting for the endpoint

### Requirement: Endpoint behavior is tested at both boundaries
The provider SHALL have hermetic tests for request construction, response parsing, cancellation, and failure classification using a local HTTP test server.
It SHALL have integration-tagged coverage for the configured authenticated endpoint.

#### Scenario: Local server verifies request shape
- **WHEN** the hermetic provider test receives a review
- **THEN** it can verify the exact URL, authorization behavior, message content, no-tool request, and non-streaming setting
