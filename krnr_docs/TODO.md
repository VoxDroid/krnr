# TODO — krnr (Kernel Runner)

This file is the canonical checklist for the project. Use `PROJECT_OVERVIEW.md` as the specification and update this file as milestones are completed.

---

## Project-level notes

- Language: Go (>= 1.22)
- CLI: Cobra
- Database: SQLite (single-file at OS-specific path)
- Cross-platform: Windows, Linux, macOS

---

## Milestones & Tasks 

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

### 3) Registry (CRUD & models) (in-progress)
- [ ] Implement `internal/registry/models.go` and `registry.go` CRUD
- [ ] Add search/list/tagging operations
- Acceptance: Unit tests for CRUD operations pass

### 4) Execution engine (cross-platform)
- [ ] Implement `internal/executor/executor.go` (interface)
- [ ] Implement `executor_unix.go` and `executor_windows.go` for shell invocation
- [ ] Implement flags: `--dry-run`, `--confirm`, `--verbose`
- Acceptance: Commands execute and stream stdout/stderr properly on Windows and Unix (tests or manual verification)

### 5) CLI commands (Cobra)
- [ ] `cmd/root.go` with base flags and config
- [ ] Implement `save`, `run`, `list`, `describe`, `edit`, `delete`, `export`, `import` commands
- Acceptance: `krnr --help` lists correct commands; each command has basic integration tests

### 6) Config & paths
- [ ] Implement `internal/config/paths.go` to resolve `~/.krnr` or Windows path
- [ ] Add config loading & init helper
- Acceptance: Config directory created and respected cross-platform

### 7) Importer & exporter
- [ ] `internal/exporter/sqlite_export.go` for exporting DB or selected command_sets
- [ ] `internal/importer/sqlite_import.go` for importing
- Acceptance: Exported artifacts are portable and import restores them

### 8) Recorder, editor & utilities
- [ ] `recorder.go` to capture interactive steps (if applicable)
- [ ] `editor.go` helper to open user editor for editing command sets
- [ ] `confirm.go` for yes/no confirmations
- Acceptance: Editing CLI flows launch editor and persist updates

### 9) Tests (unit & integration)
- [ ] Add unit tests for DB, registry, executor, and CLI
- [ ] Add integration tests that exercise save→run→list flows
- Acceptance: Tests run locally and pass with `go test ./...`

### 10) Linting & formatting
- [ ] Add `.golangci.yml` and configure IDE instructions
- [ ] Enforce `gofmt`/`goimports` and `golangci-lint` checks
- [ ] Add `scripts/lint.sh` and `fmt.sh`
- Acceptance: Codebase passes linters and format checks

### 11) Pre-commit hooks & CI
- [ ] Add `.pre-commit-config.yaml` and instructions (`pip install pre-commit`)
- [ ] Add GitHub Actions workflow for build/test/lint/cross-build
- Acceptance: Hooks prevent bad commits; CI green for PRs

### 12) Documentation
- [ ] `docs/architecture.md` (explain components)
- [ ] `docs/database.md` (schema & migration guide)
- [ ] `docs/roadmap.md` (priority features)
- Acceptance: Docs provide enough info to onboard new devs

### 13) Cross-platform build & packaging
- [ ] Add `scripts/build.sh` for cross-compile targets (windows/linux/macos)
- [ ] Document release process in `README.md`
- Acceptance: Binaries built for all platforms

### 14) Security & safety
- [ ] Confirm no background auto-runs; document safety defaults
- [ ] Add checks to prevent dangerous operations by default
- Acceptance: Security checklist reviewed and documented

### 15) Final polish & release
- [ ] Versioning, changelog, and tag (semver)
- [ ] Publish binaries / provide install instructions
- Acceptance: v1.0 release candidate ready

---

## Notes & conventions

- Follow `PROJECT_OVERVIEW.md` as the authoritative spec and update this TODO when milestones are completed.
- Keep tasks small and testable; update statuses and add subtasks as you progress.

---

_Last updated: 2026-01-06_
