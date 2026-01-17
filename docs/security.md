# Security & Safety

krnr defaults to **safe execution**: it will not run obviously destructive commands unless explicitly overridden. This page documents the project's security and safety hardening goals and a short checklist of actions that have been implemented and planned.

## Key behaviors (implemented)

- `krnr run` performs a conservative check against a list of dangerous command patterns (e.g., `rm -rf /`, `dd if=`, `mkfs`, fork bombs). If a command looks unsafe the run will be refused with a message.
- Use `--force` to override the safety check when you have verified the command is intentional.
- Use `--dry-run`, `--confirm`, and `--verbose` to preview execution without performing risky operations.
- `krnr delete` prompts interactively by default and accepts `--yes` to skip prompts for automation.
- `krnr install` prints a detailed plan and requires confirmation (or `--yes`) before modifying file system state or persistent PATH values.
- Parameter values that look like secrets (names such as `token`, `secret`, `password`, etc.) or that are supplied from environment variables are redacted in CLI output and dry-run/verbose prints.

## Checklist (what we did)

- [x] Added conservative safety checks in `internal/security` to catch obviously destructive patterns and block them by default.
- [x] Added `--force` to allow override in trusted automation contexts.
- [x] Ensured `krnr delete` and `krnr install` require explicit confirmation by default and support `--yes` for non-interactive usage.
- [x] Added parameter redaction so secrets from `--param` (env-bound or secret-looking names) are replaced with `<redacted>` in dry-run and printed output.
- [x] Added unit and CLI tests to exercise destruction-blocking, prompt behavior, and redaction (`cmd/*_test.go`, `internal/security/*_test.go`).

## Planned / follow-ups

- [ ] Add a SAST and dependency scan job to CI (e.g., GitHub CodeQL, `gosec`, `govulncheck`) and make it required for PRs.
- [ ] Add pre-commit secret detection to catch accidental commits of tokens/credentials (e.g., `pre-commit` hooks or a GitHub action that scans diffs).
- [ ] Expand the threat model and add a short remediation guide for common issues (supply-chain, privilege escalation, local file corruption).
- [ ] Add tests / fuzzers that attempt to inject dangerous patterns via saved commands to validate filtering and redaction over time.

## Usage guidance

- Prefer `--dry-run` and `--confirm` when running untrusted or shared command sets.
- Avoid storing secrets directly in saved commands; prefer environment-bound parameters (`--param name=env:VAR`) or external secret stores.
- For automation (CI), use `--force` only in controlled runners and ensure CI secrets are protected by the platform.

---
