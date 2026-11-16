## ADDED Requirements

### Requirement: Application build identity
The CLI SHALL expose `--version` and MCP SHALL report the same granular application version. Source archive releases SHALL embed the selected release version; Git builds SHALL include available revision and modified-state metadata. Application versions SHALL remain independent of protocol and output schema versions.

#### Scenario: Source archive installation
- **WHEN** a release source archive is built without Git metadata
- **THEN** the application reports its stamped release version

#### Scenario: Development checkout
- **WHEN** a binary contains Go VCS build metadata
- **THEN** reported application identity includes its revision and dirty state when modified
