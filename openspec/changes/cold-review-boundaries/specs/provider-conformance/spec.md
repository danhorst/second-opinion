## ADDED Requirements

### Requirement: Absolute-path cold evidence
The shared suite SHALL check that a reviewer cannot read a caller-owned file referenced by its absolute path but not included in material. The check SHALL be gated by the same positive control as relative-path and instruction-file checks. Invalid nil results SHALL fail checks rather than panic.

#### Scenario: Relative isolation is insufficient
- **WHEN** a reviewer can read an absolute-path canary despite an isolated working directory
- **THEN** the absolute-path cold check fails even if the relative-path check passes

#### Scenario: Failed positive control
- **WHEN** material delivery has not passed the positive control
- **THEN** the absolute-path check is inconclusive and not reported as passing
