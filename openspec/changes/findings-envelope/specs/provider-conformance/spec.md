# provider-conformance

## ADDED Requirements

### Requirement: Conformance proves byte-exact raw passthrough
The suite SHALL verify that an adapter returns reviewer output unaltered — no trimming beyond what the transport itself requires, no re-wrapping, no reordering, no injected headers.
Raw passthrough is load-bearing because the engine parses this text, so an adapter that mangles it silently changes what findings the engine can recover.

#### Scenario: Adapter passes reviewer output through unchanged
- **WHEN** the suite runs a review whose reviewer output contains multiple lines, leading and trailing whitespace, and text resembling the finding-header convention
- **THEN** the result's raw findings field matches the reviewer's output byte-for-byte

#### Scenario: Mangling adapter fails the suite
- **WHEN** the suite runs against a reference provider rigged to reformat reviewer output
- **THEN** the passthrough check fails

### Requirement: Conformance does not assert model behavior
The suite SHALL NOT require that a provider's reviewer emit output conforming to the finding-header convention.
Conformance tests adapters, not models; whether a reviewer follows the convention is nondeterministic and is recorded as uncovered rather than asserted.

#### Scenario: Unparseable real output still conforms
- **WHEN** a real reviewer returns output that the engine's parser cannot decompose into findings
- **THEN** the adapter still passes conformance, because the contract it owns is passthrough and provenance
