## Decisions

Use a merge commit based on current main, retaining the released branch as ancestry. Keep strict cold invocation, terminal-result validation and bounded capture while adding main's model extraction/material provenance and OpenAI-compatible registry entry. HTTP capture must honor MCP budgets rather than inherit a small diagnostics-only truncation limit. Shared MCP wire format retains its material receipt.

Release preflight fetches origin/main and checks remote tags, requires branch main, a clean worktree and HEAD equal to origin/main, validates a new vX.Y.Z tag before changes. Release preparation creates one commit and annotated tag and pushes main/tag atomically without force. A concurrent remote update rejects both refs.

The workflow checks out full history, validates tag syntax, verifies it resolves to HEAD, fetches main and requires tag ancestry, and checks embedded application version before building/publishing. Use environment arguments rather than interpolation into shell code. Published tags remain immutable; merged fixes require a subsequent explicitly authorized release.

## Cold-reviewer guarantee

No content assembly changes. Reconciliation retains stricter published adapter controls and marks missing real-provider evidence rather than weakening isolation to obtain passing tests.

## Validation

Real temporary Git repositories exercise wrong branch, dirty tree, ahead/behind/diverged main, existing tags, valid main, off-main publication and version mismatch. Run regression/race/build checks, then PR CI before merge and verify both released tags are ancestors of remote main.
