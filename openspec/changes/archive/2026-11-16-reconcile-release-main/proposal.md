## Why

v0.2.0 and v0.2.1 were tagged from a divergent feature branch. Main lacks released MCP work, while the released branch lacks main's provider changes. Release automation allowed this discrepancy.

## What Changes

Merge both histories without changing published tags. Preserve provider capabilities and MCP limits/provenance. Require clean synchronized main for new releases, atomic main/tag push, and an independent main-ancestry publishing gate. Document the process.

## Capabilities

### New Capabilities
- `release-process`: main-only release preparation and publication validation.

## Non-goals

Rewriting published tags, replacing released artifacts, publishing a new patch release, or claiming new real-provider certification.

## Impact

Conflict reconciliation, release scripts/workflow, executable Git-fixture tests, and release documentation.
