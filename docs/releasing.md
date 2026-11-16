# Releasing second-opinion

Release from `main` only. A user request to release never means skipping the merge.

1. Merge the intended feature work through a reviewed PR, preserving published-tag ancestry when reconciling release branches. Wait for CI.
2. Switch to `main`, fetch, and fast-forward to `origin/main`. Commit the changelog entries and any release preparation, push them, and wait for CI. `main` must be clean and exactly equal to the remote.
3. Run `scripts/release vX.Y.Z`. It fetches current main, rejects a dirty/detached/feature/ahead/behind checkout and existing tag, stamps the version and changelog, commits, creates an annotated tag, and pushes `main` and the tag atomically without force.
4. Watch the Release workflow through tests, GitHub release creation, and Homebrew formula update. Verify the public release and formula version. Publishing is not complete at tag push.

The workflow separately validates the tag name, checkout, main ancestry, and embedded version before publication. It accepts an older tag for a deliberate retry only if that tag is still on main. Do not use retries to downgrade the Homebrew formula.

If remote main advances before the atomic push, neither release ref is published. Do not force-push. Inspect local versus remote refs and reconcile the unpublished release commit before trying again. If publication failed after the tag was pushed, leave the published tag fixed and retry its workflow after diagnosing the failure. Code fixes require a new patch version.

GitHub HTTPS credentials need repo and workflow permissions. When macOS Keychain supplies a stale token before the refreshed GitHub CLI token, use a one-command empty credential helper followed by `gh auth git-credential`; do not weaken release checks or print credentials.

## Historical discrepancy

v0.2.0 and v0.2.1 were incorrectly published from `provider-discovery-and-envelope-specs` before it was merged to main. The reconciliation merges that history and retains main's independently developed provider changes. Existing tags and release artifacts remain unchanged. Main resumes development at 0.2.2-dev; this reconciliation does not publish another release. Recorded provider and live-harness verification gaps still apply.

Repository administrators can additionally restrict tag creation using GitHub rulesets. The checked-in guards protect the supported release path; scripts in old immutable tags cannot be retroactively changed.
