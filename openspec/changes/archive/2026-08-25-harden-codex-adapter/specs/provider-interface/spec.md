# provider-interface

## MODIFIED Requirements

### Requirement: Review result carries reviewer provenance
Every review result SHALL include provenance identifying the provider, the model that performed the review, the identity of the prompt used, and the identity of the material reviewed.
Provenance MUST survive to the caller unaltered so that a reviewer-equals-author violation and a material mismatch are detectable after the fact.

#### Scenario: Provenance on a successful review
- **WHEN** a provider completes a review
- **THEN** the result names the provider, the model that ran, the prompt identity, and the SHA-256 identity of the exact material value received by the provider

#### Scenario: Reported model reflects what actually ran
- **WHEN** a provider substitutes a different model than requested (for any provider-internal reason)
- **THEN** the result's provenance names the model that actually performed the review

#### Scenario: Material identity is transport-independent
- **WHEN** two providers receive byte-identical material
- **THEN** both results report the same material identity regardless of whether the providers use a subprocess, HTTP, or another transport
