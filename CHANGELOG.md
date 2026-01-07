# Changelog

All notable changes to this project will be documented in this file.

## Unreleased - 2026-01-08

- Implement editor helper tests and interactive edit behavior:
  - Add `OpenEditor` unit tests that run a scripted editor via `$EDITOR` (cross-platform).
  - Add integration test for `krnr edit` interactive flow to ensure comments (`#`) and blank lines are ignored and only non-empty lines are saved as commands.
  - Update CLI and architecture docs to describe interactive edit behavior and testability.

- Add interactive recorder:
  - `internal/recorder` provides `RecordCommands` and `SaveRecorded` to capture commands from stdin and save them into the registry.
  - Add `krnr record <name>` CLI command to record commands from stdin and persist them as a named command set.
  - Add unit test for `RecordCommands` and integration test for `krnr record`.

- Add E2E release test and CI workflow:
  - Add `internal/release/release_test.go` which runs `scripts/release.sh` in a sandbox and verifies artifacts and checksums are created.
  - Add `.github/workflows/e2e-release.yml` (manual `workflow_dispatch`) to run the E2E build.
  - The workflow uploads `dist/` as a workflow artifact for inspection by maintainers.

- CI: add a `test-matrix` job to run `go test ./...` on Ubuntu, macOS, and Windows runners to catch platform-specific failures and improve coverage.
## v0.1.0 - 2026-01-06

- Initial release: core features (save, run, list, describe, edit, delete), database, registry, executor, CLI, importer/exporter, CI, linting, release automation, and security checks.

- Initial project scaffolding and core features
- Database, registry, executor, CLI commands, importer/exporter
- CI, linting, formatting, and release automation
- Security safety checks and docs

- Tests: added strict unit tests for `run` flags (`--suppress-command`, `--show-stderr`, `--dry-run`, `--verbose`, `--confirm`, `--force`) and improved executor testability via an injectable `executor.Runner` interface.
- CI: tests run under the `test` job on all pushes and PRs (ensures new behavior is validated in CI).
