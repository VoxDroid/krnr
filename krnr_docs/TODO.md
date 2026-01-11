# TODO — krnr (Kernel Runner)

This file is the canonical checklist for the project. Use `PROJECT_OVERVIEW.md` as the specification and update this file as milestones are completed.

---

## Project-level notes

- Language: Go (>= 1.25.5)
- CLI: Cobra
- Database: SQLite (single-file at OS-specific path)
- Cross-platform: Windows, Linux, macOS

---

## Outstanding / To do (newest at top)

### Medium-term (v1.0) — Prioritized TODOs
- Tagging & Search UI
  - [x] Add CLI commands: `krnr tag add|remove|list` and integrate `--tag`/`--filter` flags for `krnr list` and `krnr search`
  - [x] Implement fuzzy search and filter helpers in `internal/registry`
  - Acceptance: Unit tests for tag attach/detach and search; CLI examples in `docs/cli.md` — **Done** (unit tests + CLI examples added)
- Parameters & Variable Substitution
  - [x] Define parameter syntax (e.g., `{{param}}`) and parser in `internal/registry`/executor
  - [x] Add runtime prompts/flags for supplying parameter values (`--param name=value`) and environment-binding support
  - Acceptance: Integration test that runs a saved command with substituted parameters — **Done** (unit tests + CLI integration test added)
- Versioning & History
  - [ ] Add version/history model for `command_sets` (DB migration and schema updates)
  - [ ] Implement `krnr history <name>` and `krnr rollback <name> --version` commands and tests
  - Acceptance: History command lists versions and rollback restores a previous version
- Packaging & Releases
  - [ ] Verify and extend `packaging/` manifests (Homebrew, Scoop, nfpm) and GoReleaser config
  - [ ] Add CI job to build and validate packaging artifacts and manifests on tag/release
  - Acceptance: Packaging artifacts generated and sample Homebrew/Scoop manifests validated in CI
- Security & Safety Hardening
  - [ ] Conduct a security review; document checklist in `docs/security.md`
  - [ ] Add explicit confirmation/prompting for destructive ops and guardrails in `cmd/delete.go` and `internal/install`
  - Acceptance: Security checklist added and tests enforce safety defaults
- Documentation & Migration Notes
  - [ ] Update `docs/roadmap.md` with status summaries and dates
  - [ ] Add migration notes for any DB schema changes in `docs/database.md`
  - Acceptance: Docs updated and referenced in the v1.0 release notes

---

## Completed / Done - Medium-term Milestone

## Completed / Done - Short-term Milestone

### Installer & release (completed)
- [x] Design installer strategy — decided scope (per-user `krnr install` CLI vs platform packages), security model, and test plan.
- [x] Implement `krnr install` — cross-platform per-user installer with `--dry-run`, `--user`, `--system`, `--path`, `--yes`, and uninstall support plus tests.
- [x] Add CI build/release jobs for installers — workflows build artifacts per platform and upload to `dist/` (GoReleaser integrated).
- [x] Test release & validate artifacts — `release-validate.yml` and artifact inspection steps added.
- [x] Add install tests — unit tests and a CI E2E `install --dry-run` smoke test.
- [x] Add uninstall/rollback and safety checks — ensure changes are reversible and prompt before editing PATH or shell rc files.
- [x] Update docs and CHANGELOG — added `docs/install.md` and updated CLI docs; documentation guidance added to README.
- [x] Add CI Windows job to run `install/uninstall --dry-run` and tests to catch PATH regressions.

### 1) Initialize repository & dev environment (completed)
- [x] Create `go.mod` and minimal package layout (`cmd/`, `internal/`)
- [x] Add `.gitignore`, `LICENSE`, and base `README.md`
- [x] Ensure Go 1.25.5+ development environment instructions in README
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
- [x] Versioning, changelog, and tag (semver)
- [ ] Publish binaries / provide install instructions
- Acceptance: v1.0 release candidate ready (changelog and release notes added; packaging/publishing remaining)

---

## Notes & conventions

- Follow `PROJECT_OVERVIEW.md` as the authoritative spec and update this TODO when milestones are completed.
- Keep tasks small and testable; update statuses and add subtasks as you progress.

---

_Last updated: 2026-01-10 (features enumerated; Windows install CI added)_
