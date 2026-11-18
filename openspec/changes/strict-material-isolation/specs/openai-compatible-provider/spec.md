## ADDED Requirements

### Requirement: Local material-only HTTP execution
The adapter SHALL advertise `local-material-only`. Its review content SHALL consist only of supplied prompt and material. It SHALL neither offer nor execute tools, dereference reviewer-provided locations, nor launch a harness. HTTP redirects SHALL fail without following Location, including with a supplied HTTP client. Responses with duplicate keys (including case-folded aliases) SHALL fail before interpretation. Responses containing nonempty tool calls or a non-null legacy function call SHALL fail without a completed review, including responses that also contain text. The configured endpoint and transport SHALL remain trusted; this capability SHALL NOT certify their internal behavior.

#### Scenario: Redirect response
- **WHEN** the endpoint returns a redirect to another location
- **THEN** the review fails and no request is made to that location

#### Scenario: Tool request with findings
- **WHEN** a response includes findings text and a tool or function call
- **THEN** the review fails without executing the request or returning findings
