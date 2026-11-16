## ADDED Requirements

### Requirement: Subprocess cleanup and bounded output drain
Subprocess adapters SHALL terminate their invocation process group on cancellation and completion on Darwin, Linux, and BSD. They SHALL bound waits for inherited output pipes to two seconds after cancellation or parent exit. On other platforms they SHALL document parent-only cancellation. Detached descendants outside the invocation group are not covered.

#### Scenario: Child holds output open
- **WHEN** a provider parent exits or is cancelled while descendants retain output pipes
- **THEN** the caller does not wait indefinitely for those pipes and supported-platform group members are terminated

### Requirement: Actionable argument-size errors
When an OS rejects provider arguments as too large, the adapter SHALL explain that material must be reduced without including material or secrets in the error, and SHALL preserve the underlying error.

#### Scenario: Oversized command arguments
- **WHEN** starting an argument-based provider fails with the OS argument-limit error
- **THEN** the result is an error with actionable size guidance and no completed review
