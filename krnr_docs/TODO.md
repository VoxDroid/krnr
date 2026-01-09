# TODO — krnr (Kernel Runner)

This file is the canonical checklist for the project. Use `PROJECT_OVERVIEW.md` as the specification and update this file as milestones are completed.

---

## Project-level notes

- Language: Go (>= 1.22)
- CLI: Cobra
- Database: SQLite (single-file at OS-specific path)
- Cross-platform: Windows, Linux, macOS

---

## Outstanding / To do (newest at top)

- [x] Design installer strategy (priority: high) — decide scope (per-user `krnr install` CLI vs platform packages), security model, and test plan. (completed)
- [x] Implement `krnr install` (priority: high) — add cross-platform per-user installer (dry-run, --user, --system, --path, --yes, --uninstall) with unit and integration tests. (completed)
- [ ] Create platform packages (priority: medium) — add GoReleaser configuration & manifests to produce Windows MSI/Winget, macOS Homebrew/cask or pkg, and Linux `.deb`/`.rpm` packages/archives. (in progress: VoxDroid)
- [x] Add CI build/release jobs for installers (priority: medium) — build artifacts for each platform and upload to `dist/` during release workflows. (goreleaser: release-goreleaser.yml added)
- [x] Test release & validate artifacts (release-validate.yml added; artifacts uploaded for inspection)
- [x] Add install tests (priority: medium) — unit tests for install logic and a CI E2E job that runs `install --dry-run` and an install->run smoke test. (completed)
- [x] Add uninstall/rollback and safety checks (priority: medium) — ensure changes are reversible and prompt before editing PATH or shell rc files. (completed)
- [x] Update docs and CHANGELOG (priority: low) — add `docs/install.md`, update `docs/cli.md` examples, and add a `CHANGELOG` entry when released. (completed)
- [ ] Add CI Windows job to run `install/uninstall --dry-run` and tests to catch PATH regressions (priority: high).
- [ ] Perform release packaging & publish manifests (priority: low) — create official Homebrew/Scoop/Winget manifests and publish in release workflow. (in progress: sample manifests added)

---

## Completed / Done (older beneath)

### 1) Initialize repository & dev environment (completed)
- [x] Create `go.mod` and minimal package layout (`cmd/`, `internal/`)
- [x] Add `.gitignore`, `LICENSE`, and base `README.md`
- [x] Ensure Go 1.22+ development environment instructions in README
- Acceptance: `go build ./...` succeeds locally on Windows

### 2) Database layer & migrations (completed)
- [x] Add `internal/db/db.go` with initialization and path discovery
- [x] Add `internal/db/schema.sql` with tables from `PROJECT_OVERVIEW.md`
- [x] Add simple migrations support (`migrations.go`)
- Acceptance: DB file created at OS path and schema validated by tests

### 3) Registry (CRUD & models)
- [x] Implement `internal/registry/models.go` and `registry.go` CRUD
- [x] Add search/list/tagging operations — implemented in `internal/registry` with `List`/`Search` helpers, tag attach/detach, and unit tests.
- Acceptance: Unit tests for CRUD and search/tagging operations pass

### 4) Execution engine (cross-platform) (completed)
- [x] Implement `internal/executor/executor.go` (implementation)
- [x] Implement `executor_unix.go` and `executor_windows.go` for shell invocation
- [x] Implement flags/options: `--dry-run`, `--confirm`, `--verbose`
- [x] Add `--shell` override and OS-aware `shellInvocation` mapping (support `pwsh`, `powershell`, `cmd`, `bash`) — implemented in `cmd/run.go` and `internal/executor` with tests and docs.
- [x] Normalize Windows shell output (`unescapeWriter`) to remove backslash-escaped quotes — implemented and tested.
- Acceptance: Commands execute and stream stdout/stderr properly on Windows and Unix (tests or manual verification)

### 5) CLI commands (Cobra)
- [x] `cmd/root.go` with base flags and config
- [x] Implement `save`, `run`, `list`, `describe` commands (basic implementations)
- [x] Implement `edit`, `delete` commands (basic implementations)
- [x] Implement `export`, `import` commands
- [x] Add `krnr record` command to capture commands from stdin (recorder)
- [x] Add `whoami` command group (set/show/clear) and `--author`/`--author-email` flags on `save` to persist author metadata
- Acceptance: `krnr --help` lists correct commands; each command has basic integration tests

### 6) Config & paths (completed)
- [x] Implement `internal/config/paths.go` to resolve `~/.krnr` or Windows path
- [x] Add config loading & init helper (`EnsureDataDir`)
- Acceptance: Config directory created and respected cross-platform

### 7) Importer & exporter (completed)
- [x] `internal/exporter/sqlite_export.go` for exporting DB or selected command_sets
- [x] `internal/importer/sqlite_import.go` for importing
- Acceptance: Exported artifacts are portable and import restores them

### 8) Recorder, editor & utilities
- [x] `recorder.go` to capture interactive steps — implemented `internal/recorder`, added `krnr record` CLI, and tests.
- [x] `editor.go` helper to open user editor for editing command sets — implemented editor helper and added integration tests that exercise interactive edits.
- [x] `confirm.go` for yes/no confirmations
- Acceptance: Editing and recording CLI flows launch editor/recording and persist updates

### 9) Tests (unit & integration)
- [x] Add unit tests for DB, registry, executor, and CLI
- [x] Add integration tests that exercise save→run→list flows
- [x] Add strict run-flag tests (`--suppress-command`, `--show-stderr`, `--dry-run`, `--verbose`, `--confirm`, `--force`)
- [x] Add E2E roundtrip integration test (save → run → export → import) — `cmd/save_run_export_import_test.go` verifies full-cycle portability
- [x] Add `cmd/save_run_export_import_test.go` and update docs to reflect E2E coverage
- Acceptance: Tests run locally and pass with `go test ./...`

### 10) Linting & formatting (completed)
- [x] Add `.golangci.yml` and configure IDE instructions
- [x] Enforce `gofmt`/`goimports` and `golangci-lint` checks via pre-commit
- [x] Add `scripts/lint.sh` and `fmt.sh`
- Acceptance: Codebase passes linters and format checks

### 11) Pre-commit hooks & CI (completed)
- [x] Add `.pre-commit-config.yaml` and instructions (`pip install pre-commit`)
- [x] Add GitHub Actions workflow for build/test/lint/cross-build (test-matrix on Ubuntu/macOS/Windows)
- [x] Add resilient `scripts/lint.sh` and tests that cover missing local linter and Docker fallback; fixed test path resolution so CI invokes the script reliably
- [x] Add E2E release test/workflow (manual dispatch) and enforce release version consistency in release workflow
- Acceptance: Hooks prevent bad commits; CI runs on PRs and builds artifacts

### 12) Documentation (completed)
- [x] `docs/architecture.md` (explain components)
- [x] `docs/database.md` (schema & migration guide)
- [x] `docs/roadmap.md` (priority features)
- [x] Update `docs/cli.md` with `--shell` usage and examples
- [x] Update `docs/ci.md` with lint fallback behavior and test notes
- Acceptance: Docs provide enough info to onboard new devs

### 13) Cross-platform build & packaging (completed)
- [x] Add `scripts/build.sh` for cross-compile targets (windows/linux/macos)
- [x] Add `scripts/release.sh` + `release.yml` workflow to package & upload releases
- [x] Document release process in `README.md`
- Acceptance: Binaries built for all platforms and packaged for releases

### 14) Security & safety (completed)
- [x] Confirm no background auto-runs; document safety defaults
- [x] Add checks to prevent dangerous operations by default
- Acceptance: Security checklist reviewed and documented (safety check + docs added)

### 15) Final polish & release
- [ ] Versioning, changelog, and tag (semver)
- [ ] Publish binaries / provide install instructions
- Acceptance: v1.0 release candidate ready

---

## Notes & conventions

- Follow `PROJECT_OVERVIEW.md` as the authoritative spec and update this TODO when milestones are completed.
- Keep tasks small and testable; update statuses and add subtasks as you progress.

---

_Last updated: 2026-01-09 (features enumerated)_
