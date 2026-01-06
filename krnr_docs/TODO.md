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

- [ ] Add search/list/tagging operations to the registry (priority: medium)
- [ ] Implement `editor.go` helper to open the user's editor for editing command sets (priority: low)
- [ ] Implement `recorder.go` if interactive recording is desired (priority: low)
- [ ] Add an E2E release test that runs on CI using `PERSONAL_TOKEN` (priority: high)
- [ ] Add CI job(s) to run tests across multiple OS runners (windows/linux/macos) to catch platform-specific behaviors (priority: medium)
- [ ] Enforce version consistency: add a check that `internal/version/version.go` matches release commit/title during the release workflow (priority: medium)
- [ ] Add tests that cover pre-commit lint fallback behavior and Docker fallback in CI (priority: low)
- [ ] Improve CLI UX: consider `krnr whoami` persistence or opt-in author metadata for saved command sets (priority: low)
- [ ] Add E2E integration tests that exercise save -> run -> export -> import roundtrip (priority: medium)

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
- [ ] Add search/list/tagging operations
- Acceptance: Unit tests for CRUD operations pass

### 4) Execution engine (cross-platform) (completed)
- [x] Implement `internal/executor/executor.go` (implementation)
- [x] Implement `executor_unix.go` and `executor_windows.go` for shell invocation
- [x] Implement flags/options: `--dry-run`, `--confirm`, `--verbose`
- Acceptance: Commands execute and stream stdout/stderr properly on Windows and Unix (tests or manual verification)

### 5) CLI commands (Cobra)
- [x] `cmd/root.go` with base flags and config
- [x] Implement `save`, `run`, `list`, `describe` commands (basic implementations)
- [x] Implement `edit`, `delete` commands (basic implementations)
- [x] Implement `export`, `import` commands
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
- [ ] `recorder.go` to capture interactive steps (if applicable)
- [ ] `editor.go` helper to open user editor for editing command sets
- [x] `confirm.go` for yes/no confirmations
- Acceptance: Editing CLI flows launch editor and persist updates

### 9) Tests (unit & integration)
- [x] Add unit tests for DB, registry, executor, and CLI
- [x] Add integration tests that exercise save→run→list flows
- [x] Add strict run-flag tests (`--suppress-command`, `--show-stderr`, `--dry-run`, `--verbose`, `--confirm`, `--force`)
- Acceptance: Tests run locally and pass with `go test ./...`

### 10) Linting & formatting (completed)
- [x] Add `.golangci.yml` and configure IDE instructions
- [x] Enforce `gofmt`/`goimports` and `golangci-lint` checks via pre-commit
- [x] Add `scripts/lint.sh` and `fmt.sh`
- Acceptance: Codebase passes linters and format checks

### 11) Pre-commit hooks & CI (completed)
- [x] Add `.pre-commit-config.yaml` and instructions (`pip install pre-commit`)
- [x] Add GitHub Actions workflow for build/test/lint/cross-build
- Acceptance: Hooks prevent bad commits; CI runs on PRs and builds artifacts

### 12) Documentation (completed)
- [x] `docs/architecture.md` (explain components)
- [x] `docs/database.md` (schema & migration guide)
- [x] `docs/roadmap.md` (priority features)
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

_Last updated: 2026-01-07_
