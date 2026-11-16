# Validation — 2026-11-16

- Four launcher/formula test cases pass, covering environment/argv and status propagation, explicit token paths, invalid lengths/permissions/symlinks/whitespace/NUL bytes, and formula substitutions.
- Bash syntax and rendered Ruby syntax pass.
- Installed Homebrew parsed the rendered formula with Formulary.from_path and generated a plist accepted by plutil. It contains RunAtLoad, the stable opt-libexec command, an explicit Homebrew/system PATH, and log destinations, with no token value or token environment entry.
- make test passes (Python packaging tests, go vet, all Go tests).
- No service was installed, started, or registered. No real credentials were created. Live launchd/systemd lifecycle and provider authentication in the service environment remain unverified per the user's machine constraint; this is recorded in AGENTS.md.
- Tap publication waits for a future release containing the new launcher; the already-published v0.2.0 formula is not changed.
