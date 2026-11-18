# strict-material-isolation

## Purpose

Give callers an enforceable local material boundary while distinguishing client guarantees from trusted endpoint behavior and model conformance.

## Requirements

### Requirement: Scoped local material boundary
The `local-material-only` boundary SHALL mean that, after caller assembly, the adapter uses only supplied prompt and material as review content and cannot service reviewer-directed filesystem reads, URL fetches, tools, or harness execution. Credentials, configuration, HTTP transport, and the selected endpoint SHALL be explicit trusted dependencies. The boundary MUST NOT claim endpoint-side context isolation, OS isolation, local-only inference, or model obedience.

#### Scenario: Ambient path in material
- **WHEN** material or a response mentions an ambient file or URL
- **THEN** a supporting adapter treats it as text and does not dereference it

### Requirement: Conservative support reporting
Providers without a recognized boundary SHALL report `unverified`. Existing cold invocation obligations SHALL remain in force. Provider discovery and successful execution results SHALL report the effective boundary without a model call for discovery.

#### Scenario: Unsupported harness
- **WHEN** a subprocess adapter has cold controls but no certified local boundary
- **THEN** it reports `unverified`

### Requirement: Evidence matches the claim
Local policy and transport tests SHALL establish enforcement mechanics without claiming real-model conformance. Live conformance SHALL retain positive controls and bounded cold probes. Unverified endpoint behavior and OS isolation SHALL be documented.

#### Scenario: Local fixture passes
- **WHEN** a local HTTP fixture confirms no follow-up tool or redirect requests
- **THEN** the evidence is reported as client enforcement testing, not endpoint or model certification
