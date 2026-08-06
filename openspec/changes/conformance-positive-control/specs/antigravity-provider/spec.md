# antigravity-provider

## ADDED Requirements

### Requirement: The instruction is placed where the harness reads it
The adapter SHALL place the composed instruction where `agy` actually reads the prompt: as the value consumed by `--print`, which takes the prompt as its argument rather than acting as a boolean.
No other flag may sit between `--print` and the instruction, because `agy` would consume that flag as the prompt and silently review nothing.

#### Scenario: Print mode receives the instruction
- **WHEN** the adapter constructs an invocation
- **THEN** the argument immediately following `--print` is the composed instruction, not another flag

#### Scenario: Forced model does not displace the instruction
- **WHEN** a model is forced
- **THEN** the model flag precedes `--print`, and the instruction still immediately follows `--print`

## MODIFIED Requirements

### Requirement: Conformance behind the integration tag
The adapter SHALL pass the shared conformance suite against the real `agy` binary, gated behind the `integration` build tag, with deterministic white-box assertions on the constructed invocation.
White-box assertions SHALL check argument *semantics*, not merely that expected flags appear somewhere in the invocation: an adapter that includes every required flag in an order the harness misreads MUST fail.

#### Scenario: White-box cold invocation
- **WHEN** the adapter constructs an agy invocation
- **THEN** a unit test can assert the sandbox flag, print mode, the composed instruction's position, and the absence of workspace grants without executing agy

#### Scenario: Flag presence alone does not satisfy the assertion
- **WHEN** an invocation contains every required flag but orders them so the harness consumes a flag as the prompt
- **THEN** the white-box assertion fails

#### Scenario: Real-binary conformance
- **WHEN** `make test-integration` runs with `agy` on PATH and an authenticated session
- **THEN** the adapter passes the full conformance suite against the real binary, including the positive control
