# Validation — 2026-11-13

See `docs/reviews/2026-11-13-hardening.md` for shared validation results and limitations.
Default tests, vet, race tests, integration-tag compilation, and Windows/Linux cross-builds pass.
Real Codex and Antigravity conformance pass, including the positive control and new absolute-path check.
Claude real-binary conformance is blocked by expired OAuth; cold checks are inconclusive, not passing.
Current specs were synced. No provider binaries were mocked, no real global configuration was altered, and no release was made.
