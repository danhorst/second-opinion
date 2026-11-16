## Why

The current provider contract prevents accidental context loading, but a subprocess-backed reviewer can still deliberately read files from its working environment.
That weakens the guarantee that the reviewer saw only the explicitly assembled material, especially when reviewing sensitive or untrusted repositories.

## What Changes

- Add a strict material-isolation capability for providers that can enforce it.
- Define a provider-level execution contract that prevents deliberate filesystem reads outside the explicitly supplied review material.
- Introduce an explicit capability or mode distinction so callers can require strict isolation rather than assuming that cold invocation provides it.
- Define failure behavior when a provider cannot satisfy strict isolation.
- Add conformance coverage that proves the reviewer cannot obtain ambient filesystem content.
- Document which providers support strict isolation and which provide only implicit-context isolation.

### Non-goals

- This change does not change the adversarial review prompt or finding format.
- This change does not make remote providers local or prevent material from leaving the machine.
- This change does not add provider ranking, same-model enforcement, or a new provider implementation unless required to demonstrate the contract.

## Capabilities

### New Capabilities

- `strict-material-isolation`: Require and verify that a provider can review explicitly supplied material without deliberate access to ambient filesystem content.

### Modified Capabilities

- `provider-interface`: Extend provider capabilities and request behavior so callers can distinguish strict isolation from implicit-context isolation and handle unsupported requirements explicitly.

## Impact

- `internal/provider`: Provider capability metadata, isolation requirements, errors, and shared conformance tests.
- Provider adapters under `internal/provider/`: Process or endpoint-specific enforcement and truthful capability reporting.
- `internal/review` and CLI surfaces: Selection or validation when strict isolation is requested.
- Integration tests and documentation: Isolation probes, provider support matrix, and security limitations.
- Potential dependencies: OS-level sandboxing or a narrowly scoped helper process, depending on the design selected for subprocess providers.
