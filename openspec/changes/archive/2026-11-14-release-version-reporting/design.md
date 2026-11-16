## Decisions

Embed a VERSION file starting at 0.2.0-dev. Append Git revision and dirty status as SemVer build metadata when Go build information supplies them. Release automation writes the selected version before committing/tagging, so source tarball builds retain their release identity without Git. CLI --version and MCP use the same function; wire schema version 1 is unchanged.

## Cold-reviewer guarantee

Build metadata contains no review input and is never added to provider requests. No cold-review behavior changes.

## Validation

Test metadata formatting, run existing CLI/MCP checks, inspect a built binary, and validate release-script syntax without publishing.
