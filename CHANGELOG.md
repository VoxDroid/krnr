# Changelog

All notable changes to this project will be documented in this file.

## Unreleased / v1.0.0 - 2026-01-11 (upcoming)

- Major: v1.0.0 release candidate.
- Key features included in v1.0.0:
  - Tagging & Search UI: `krnr tag add|remove|list` and `--tag` filtering for `krnr list`.
  - Text search and fuzzy search: `--filter` (substring search) and `--fuzzy` (case-insensitive subsequence matching) implemented in `internal/registry`.
  - Parameters & Variable Substitution: `--param` flag with `{{param}}` syntax, interactive prompting, and `env:VAR` support implemented in `krnr run`.
  - Unit tests and integration tests for tags, search, fuzzy matching, and parameter substitution (CLI + repository tests).
  - CLI docs updated (`docs/cli.md`) with examples for `--tag`, `--filter`, and `--fuzzy`.
  - Stability, packaging, and security checks validated; cross-platform tests added/updated (Windows, Linux, macOS).
  - Refactor: reduced cyclomatic complexity for several large functions (`internal/install.Uninstall`, `internal/install.addToPath`, `internal/install.GetStatus`, `internal/recorder.RecordCommands`, `internal/importer.ImportCommandSet`) and split large tests in `internal/registry` to improve maintainability and test isolation.
- Upgrade notes:
  - No DB schema changes are required for tagging and fuzzy search — backward compatible with prior versions.
- Acceptance criteria:
  - Unit/integration tests added and passing locally; CLI examples documented in `docs/cli.md`.

## v0.2.1 - 2026-01-10

- Packaging & release automation improvements:
  - Integrate GoReleaser into the release workflow to produce installer artifacts (Deb/RPM via `nfpm`, Homebrew formula, Scoop/Winget manifests, and Windows archives).
  - Add initial `.goreleaser.yml` and sample packaging manifests under `packaging/` (Homebrew, Scoop, Winget) and `packaging/nfpm/` (deb/rpm).
  - Add `release-validate.yml` workflow to run GoReleaser in snapshot mode and upload generated `dist/` artifacts for inspection before publishing.
  - Extend `release.yml` to create portable archives and run GoReleaser to publish artifacts when a version commit is detected.
  - Document publishing notes and required token scopes in `docs/install.md` (use `PERSONAL_TOKEN` for cross-repo pushes and package registry publishing).

- Documentation & CI polish:
  - Add validation steps and documentation for packaging and publishing in `docs/install.md`.
  - Add TODO and docs entries to track packaging progress and manifest publishing.


## v0.2.0 - 2026-01-09

- Implement editor helper tests and interactive edit behavior:
  - Add `OpenEditor` unit tests that run a scripted editor via `$EDITOR` (cross-platform).
  - Add integration test for `krnr edit` interactive flow to ensure comments (`#`) and blank lines are ignored and only non-empty lines are saved as commands.
  - Update CLI and architecture docs to describe interactive edit behavior and testability.

- Add interactive recorder:
  - `internal/recorder` provides `RecordCommands` and `SaveRecorded` to capture commands from stdin and save them into the registry.
  - Add `krnr record <name>` CLI command to record commands from stdin and persist them as a named command set.
  - `krnr record` now detects when the provided `<name>` already exists and will warn and prompt the user to enter a different name instead of failing with a DB constraint error.
  - `krnr delete <name>` now prompts interactively for a y/n confirmation by default; use `--yes` to skip prompts for non-interactive use.
  - `krnr record` supports sentinel stop commands: type `:end` on a line by itself to stop recording immediately; `:save` and `:quit` are accepted as aliases. This is documented in `docs/cli.md`.
  - Docs: add a short "Install & setup" quick-start to `docs/install.md` and a summary install note in `README.md` to help users get started.
  - Add unit test for `RecordCommands` and integration test for `krnr record`.

- Add E2E release test and CI workflow:
  - Add `internal/release/release_test.go` which runs `scripts/release.sh` in a sandbox and verifies artifacts and checksums are created.
  - Add `.github/workflows/e2e-release.yml` (manual `workflow_dispatch`) to run the E2E build.
  - The workflow uploads `dist/` as a workflow artifact for inspection by maintainers.

- CI: add a `test-matrix` job to run `go test ./...` on Ubuntu, macOS, and Windows runners to catch platform-specific failures and improve coverage.

- Release workflow: enforce version consistency by verifying that the version found in the release commit message matches `internal/version/version.go`; the workflow will fail if they do not match.

- Tests: add unit/integration tests for the `scripts/lint.sh` fallback behavior (local missing, export-data error, and Docker fallback).

- CLI UX: add `whoami` persistent identity (set/show/clear) and opt-in author metadata for saves (`--author`, `--author-email`). The registry stores `author_name`/`author_email` on command sets and the DB migration ensures columns are added on upgrade.

- Add `--shell` flag to `krnr run` to allow explicitly selecting the shell (e.g., `pwsh`, `powershell`, `cmd`, `bash`) so platform-specific commands (PowerShell cmdlets) can be executed as intended. Behavior is OS-aware: unspecified shell uses sensible defaults (`cmd` on Windows, `bash` on Unix-like systems); `--shell powershell` prefers the Windows `powershell` executable when available and falls back to `pwsh` if present. Added unit and CLI tests and updated `docs/cli.md` to document usage and examples (see `cmd/run.go` and `internal/executor/shellInvocation`).

## Installer & Windows PATH fixes

- Add `krnr install` and `krnr uninstall` commands:
  - `krnr install` supports `--user` (default), `--system` (requires elevation), `--path`, `--from`, `--yes`, and `--dry-run`.
  - `krnr uninstall` reads recorded install metadata and attempts to restore previous PATH values; supports `--dry-run`, `--yes`, and `--verbose` (prints before/after PATH diagnostics).
  - `krnr install --dry-run` and `krnr uninstall --dry-run` show planned actions without performing them.

- Windows PATH handling hardened:
  - Use PowerShell `-EncodedCommand` (UTF-16LE base64) to avoid quoting/escaping issues when writing PATH.
  - Both install and uninstall run a post-write normalization fixer that collapses doubled backslashes and corrects PATH corruption (resolves an observed issue where uninstall left doubled backslashes). The fixer runs after install, after a full restore, and after removing individual entries.
  - Tests and CI use `KRNR_TEST_NO_SETX=1` to avoid persisting PATH changes in test environments.

- Add `krnr status` command that detects user/system installs and whether the user path is on PATH (useful for CI and diagnostics).

- Added `PlanUninstall()` and CLI `--dry-run` behavior so users can preview uninstall actions safely before performing them.

## v0.1.0 - 2026-01-06

- Initial release: core features (save, run, list, describe, edit, delete), database, registry, executor, CLI, importer/exporter, CI, linting, release automation, and security checks.

- Initial project scaffolding and core features
- Database, registry, executor, CLI commands, importer/exporter
- CI, linting, formatting, and release automation
- Security safety checks and docs

- Tests: added strict unit tests for `run` flags (`--suppress-command`, `--show-stderr`, `--dry-run`, `--verbose`, `--confirm`, `--force`) and improved executor testability via an injectable `executor.Runner` interface.
- CI: tests run under the `test` job on all pushes and PRs (ensures new behavior is validated in CI).
