## Decisions

Use 32 random bytes encoded as 64 hexadecimal characters and one trailing newline. Default to ~/.config/second-opinion/mcp-token; --token-file overrides SECOND_OPINION_MCP_TOKEN_FILE. Create missing directories with 0700 and the token with 0600. Publish a fully written temporary file through a same-directory hard link, atomically refusing replacement. Concurrent init callers validate the winner. Remove the temporary name on exit. Never chmod or replace an existing token. Validate regular nonsymlink file, current-user ownership, mode 0400/0600, and the launcher-compatible 32–4096-character single-line bearer syntax. Reject root and platforms outside macOS/Linux, matching service support.

Existing tokens are opened and compared against their lstat identity before bounded reading; failures produce safe diagnostics. No --force or --show-token flag. Output names the credential path, loopback endpoint, and next steps; custom paths need matching service-manager configuration. Parent directory permissions already chosen by the user are not silently changed.

## Cold-reviewer guarantee

Only deployment authentication configuration is created/read. No provider invocation or repository/client context enters a review.

## Validation

Test new/existing tokens, entropy/format, modes, symlinks, malformed and oversized files, custom paths, unknown arguments, concurrency, and no secret output. No real user credential file or service is touched.
