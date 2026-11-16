# release-process

## Purpose

Ensure published versions are validated main history and preserve immutable tags.

## Requirements


### Requirement: Main-only release preparation
Release tooling SHALL reject any branch other than main, any dirty working tree, any HEAD different from freshly fetched origin/main, and any pre-existing version tag before modifying files or creating refs. Release commit and tag SHALL be pushed atomically without force.

#### Scenario: Feature branch release attempt
- **WHEN** release preparation runs on a feature branch
- **THEN** it fails before changing version, changelog, commits, or tags

#### Scenario: Remote advancement
- **WHEN** remote main changes before the release push
- **THEN** the non-force atomic push fails without publishing the release tag

### Requirement: Independent publication guard
The publishing workflow SHALL validate the requested tag syntax, verify checkout identity and embedded version, and require the tag commit to be reachable from freshly fetched main before publishing a release or changing the tap. Published tags SHALL NOT be moved to resolve branch discrepancies.

#### Scenario: Off-main tag
- **WHEN** a tag points at a commit absent from main history
- **THEN** publication fails before release or tap changes
