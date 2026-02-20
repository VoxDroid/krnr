# Changelog


All notable changes to this project will be documented in this file.

## v1.2.9 - 2026-02-20

- **Bugfix (TUI/Status):** Do not report `on PATH:true` for an installation scope simply because the directory appears on PATH — the `krnr` binary must exist at the expected location (or be resolvable) for `GetStatus`/TUI to report `on PATH:true`. This prevents false-positive status reporting in the TUI and CLI.
- **Tests:** Added unit test to verify directories-on-PATH without the binary do not cause `on PATH` to be reported.
- **Docs:** Clarified status semantics in `docs/install.md` and release notes.
- **Version:** Bump to `v1.2.9`.

## v1.2.8 - 2026-02-20

- **Bugfix (Installer):** Fix several sudo/user install edge-cases so installs recorded under `sudo` behave correctly for the original user:
  - Record install metadata into the invoking user's data directory (SUDO_USER) and ensure the metadata file is readable by that user so `krnr status` and `krnr uninstall` work without sudo.
  - Treat recorded `AddedToPath` as authoritative for status reporting so `krnr status` shows `on PATH: true` immediately after an add-to-PATH install (even before a new shell session).
  - When running a "user" install under `sudo`, chown the installed binary and the `bin` directory to the target user so later non-sudo `uninstall` succeeds.
  - Fix reinstall-after-uninstall behavior for sudo→user installs so status and metadata remain consistent.
- **Tests:** Added unit tests covering SUDO_USER install/uninstall/reinstall flows and metadata-based status inference.
- **Docs:** Documented sudo→user installer semantics in `docs/install.md`.
- **Version:** Bump to `v1.2.8`.

## v1.2.7 - 2026-02-19

- **Bugfix (Executor/TUI):** Prevent passwords from being visible when running interactive commands (e.g., `sudo`). When the executor runs a child in hybrid PTY mode we now temporarily disable *local echo* on the caller's terminal so typed passwords are not echoed back to the host terminal. Only the local ECHO flag is toggled (other terminal output processing is preserved) to avoid changing how child output is rendered.
  - Added `makeRaw`/`restoreTerminal` hooks and OS-specific `setEcho` helpers to safely toggle host echo and make the behavior testable.
  - Added `TestExecute_SetsHostTerminalRaw` and PTY-simulation helpers to prevent regressions.
- **Quality:** Fixed revive lint warnings and kept `gocyclo` under the configured threshold; all `golangci-lint`, `gocyclo`, and unit tests pass locally.
- **Docs:** Updated `docs/executor.md` and `docs/tui.md` to document the host-terminal echo behavior during interactive PTY-backed runs.
- **Version:** Bump to `v1.2.7`.


## v1.2.6 - 2026-02-17

- **Bugfix (Registry):** Fix rollback creating duplicate version entries — rolling back to a previous version was producing both an "update" and a "rollback" version record because `ApplyVersionByName` called `ReplaceCommands` (which records an "update") and then separately recorded a "rollback". Now uses a single transaction with `replaceCommandsTx` + `recordVersionTx` so only one "rollback" version is created.
- **Bugfix (TUI):** Fix version preview not updating after a run — stale run logs were overriding the left panel content when switching panes or pressing Enter on a version. Now clears run logs on any pane navigation so: pressing Enter on a version shows its detail in the left panel, switching to the versions pane shows the version preview, and switching back restores the original command set detail.
- **Bugfix (Tests):** Fix flaky test failures caused by SQLite lock contention when multiple test packages run in parallel against the same database file. Added `setupTestDB` helper that isolates each test to its own temporary database via `KRNR_DB` env var. Applied to all registry, model, and adapter tests.
- **Portability:** Fix Windows cross-compilation — split Unix-only PTY code (`syscall.SysProcAttr.Setsid`/`Setctty`, `creack/pty`) into build-tagged files (`pty_unix.go`, `pty_windows.go`). Windows stubs return safe defaults. Added `!windows` build tags to PTY-dependent test files.
- **Version:** Bump to `v1.2.6`.

## v1.2.5 - 2026-02-17

- **Feature (TUI/Executor):** Interactive commands (e.g., `sudo`, `pacman`) now work correctly in the TUI. Prompts appear inside the run output viewport (not the footer) and user input is forwarded to the running process.
  - **Hybrid PTY:** The executor uses a hybrid PTY approach — the child's stdin and controlling terminal use a PTY (so programs that open `/dev/tty` like `sudo` work), while stdout/stderr remain as pipes (so programs like `fastfetch` detect pipe mode and produce simple, viewport-friendly output).
  - **Live streaming:** The non-PTY executor path now streams output live via `io.MultiWriter` instead of buffering until command completion. Output appears immediately in the viewport without requiring a keypress.
  - **Viewport fix:** The detail view now shows run logs during/after a run instead of unconditionally rendering static detail text, which was overwriting live output on every render frame. Logs are cleared on navigation back.
  - `Executor.Execute` accepts an explicit `stdin io.Reader`; the TUI adapter wraps it with the host terminal fd to trigger hybrid PTY mode.
- **Bugfix (TUI):** Fix doubled output when stdout and stderr point to the same writer in PTY mode. Added `copyPTYOutput` helper (later replaced by hybrid PTY).
- **Bugfix (TUI):** Fix UI border corruption caused by escape sequences split across read boundaries. Added `trailingIncompleteEscape` buffering across chunk reads.
- **Bugfix (Executor):** Fix `panic: nil pointer dereference` in `writeOutputs`/`checkExecutionError` when the PTY starter returned nil buffers. Now returns `&bytes.Buffer{}` on error and guards nil buffers.
- **Bugfix (TUI):** Fix truncated output for commands like `fastfetch` — increased read buffer from 256 to 4096 bytes and flush the escape-sequence carry buffer on EOF.
- **Bugfix (Sanitizer):** Improved CSI sequence handling — cursor-forward (`CUF`/`\x1b[nC`) is converted to spaces, cursor-horizontal-absolute (`CHA`/`\x1b[nG`) to a separator, and SGR color codes are preserved. Other destructive CSI sequences are stripped.
- **Bugfix (TUI):** Fix command list appearing empty on launch until a resize or keypress. `Init()` was mutating the model inside an async `tea.Cmd` and returning `nil` instead of a proper `tea.Msg`, so `Update()` never triggered a re-render. Now returns an `initDoneMsg` processed by `Update()` to populate the list on the first frame.
- **Bugfix (TUI):** Fix run output not visible in detail view when command has version history — the versions-panel rendering path unconditionally overwrote the viewport with static detail text, hiding run logs.
- **Bugfix (TUI):** Fix inability to scroll run output after a run completes — `GotoBottom()` was called on every render frame when logs existed, resetting scroll position. Now only auto-scrolls while the run is actively in progress.
- **Feature (TUI):** Enter on versions panel now loads the selected version's full detail into the left preview pane and switches focus there.
- **Feature (TUI):** Delete version (`D`) — pressing `D` on a highlighted version in the versions panel prompts for confirmation and deletes that specific version record. Added `DeleteVersionByName` to registry, adapter, and model layers.
- **Quality:** Added `gocyclo` (threshold >10) to pre-commit hooks; refactored `executorAdapter.Run` and `TestExecute_PTYInteractive` to reduce cyclomatic complexity below 10. All `golangci-lint` and `gocyclo` checks pass.
- **Tests:** Added tests for escape-sequence splitting, EOF flush, fd-reader wrapping, non-terminal stdin, PTY simulated behavior, sanitizer (SGR, alt-screen, CR, CUF, CHA), and headless prompt-in-viewport tests.
- **Docs:** Updated `docs/executor.md`, `docs/tui.md`, and `docs/cli.md` to describe hybrid PTY behavior, live streaming, and interactive prompt support.
- **Version:** Bump to `v1.2.5`.

## v1.2.4 - 2026-02-04

- **Bugfix (TUI):** Prevent TUI deformation when running commands that emit control sequences (e.g., `fastfetch`, `htop`, or other terminal programs that use alternate screen or cursor controls). We now sanitize run output shown in the TUI so destructive control sequences (alternate-screen, clear-screen, cursor movement, OSC sequences) are removed while preserving SGR color sequences so colored output still renders in the output pane.
  - Added `internal/tui/sanitize.RunOutput` to perform conservative sanitization and normalization.
  - Wire sanitizer into streaming path: `internal/tui/adapters/executor_adapter.go` and `cmd/tui/ui/tui.go` so run output cannot escape the output viewport.
  - Add tests: headless TUI test `TestRunStreamsSanitizesControlSequences`, and unit tests for the sanitizer (`internal/tui/adapters/executor_adapter_test.go`).
- **Quality:** Ran `gocyclo` and addressed high complexity issues (split large headless test). Fixed `golangci-lint` findings. All tests pass locally.
- **Docs:** Note about sanitized run output added to `docs/tui.md` and release notes added for v1.2.4.

## v1.2.3 - 2026-02-03

- **Bugfix (TUI):** Fix spacebar handling in the TUI editor and filter. Some terminals emit `tea.KeySpace` instead of `tea.KeyRunes` for the spacebar; the TUI now treats `KeySpace` as a typed space so spaces are inserted into editor commands and list filters consistently. Added headless tests `TestEditorTypingSpaceKeyInCommands` and `TestFilterModeSpaceKeyAppendsSpace` to prevent regressions.
- **Quality:** Ran `gocyclo` (no functions with complexity > 10 found) and `golangci-lint` (0 issues).
- **Docs:** Update `docs/tui.md` and add release notes for the fix.

## v1.2.2 - 2026-01-17

- Bugfix: Preserve comma-containing commands passed via a single `-c` flag (e.g., PowerShell `Select-Object` usage such as `Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture`). The CLI no longer splits a single `-c` value on commas — instead we use repeatable `-c` flags (`StringArray`) and ensure a single quoted `-c` is preserved literally.
- Internal: Switch `-c` command flags from `StringSlice` (which splits on commas) to `StringArray` to avoid unintended splitting; update retrieval functions accordingly and add unit tests to validate behavior.
- Tests: Add `TestSaveCommand_PreservesCommasInFlag` and update save-related tests to be robust across Windows-style inputs and quoting edge cases.
- Docs: Update `docs/cli.md` and add release notes for v1.2.2 describing the change and upgrade guidance.

## v1.2.1 - 2026-01-17

- CLI: **save** — Enhanced the `krnr save` command to better handle shell quoting mistakes by joining or merging split command arguments (e.g., when shells split embedded quotes) and heuristically reinserting `/C:"pattern"` quoting for common `findstr` usages. Adds robust argument splitting via `github.com/kballard/go-shellquote` and new unit tests.
- Executor: On Windows, the executor now directly handles simple `A | findstr ...` pipelines by executing the left command and piping its stdout into `findstr` without invoking `cmd.exe`, avoiding fragile cmd parsing/quoting issues.
- Tests & Quality: New tests covering save/merge behavior and Windows `findstr` pipeline execution added; cyclomatic complexity and linter issues addressed.
- Docs & Release Notes: Update `docs/cli.md`, `docs/releases/v1.2.1.md`, and `RELEASE_NOTES/GITHUB_v1.2.1.md` to document these changes.

## v1.2.0 - 2026-01-17

- UI: **TUI initial release (v1.2.0)** — The Terminal UI (`krnr tui`) is now available as an interactive entrypoint. Initial release focuses on delivering interactive parity for common CLI workflows while keeping the CLI as the canonical, scriptable surface.
  - Included features: browse (`list`) with filtering & fuzzy search, preview & full-screen details (`describe`), `run` with parameter prompts and streaming logs viewport, save/create/edit modals, import/export helpers (set & db), history viewing and rollback, installer views (`install`/`uninstall` dry-run planning), and `status` diagnostics.
  - Implementation: `cmd/tui` (Bubble Tea model) and `internal/tui` adapters reuse existing core packages (`registry`, `executor`, `importer`, `exporter`) and avoid duplicating business logic.
  - Tests & CI: Headless UI unit tests and PTY-based E2E tests added; CI includes headless TUI checks and PTY E2E runs where supported.
  - Docs & packaging: `docs/cli.md` documents `krnr tui`; `krnr_docs/TUI_MILESTONE.md` and `docs/releases/v1.2.0.md` updated; release notes and packaging guidance added to `RELEASE_NOTES/GITHUB_v1.2.0.md`.
  - Acceptance: Users can perform common interactive workflows via `krnr tui` with parity for interactive flows; the CLI remains the authoritative automation interface.

## v1.1.0 - 2026-01-12

- UX: `krnr save` will now detect when the provided name already exists and will prompt the user to enter a different name interactively (mirrors `krnr record` behavior). This prevents a DB constraint error when saving a duplicate name and improves CLI consistency.
- Tests: add integration tests for `krnr save` prompting on duplicate names and normal save behavior.
- Docs: update `docs/cli.md` to document the interactive duplicate-name prompt for `krnr save`.
- Add `krnr export` and `krnr import` commands to export/import the full DB or single named command sets. Exported sets are portable SQLite files usable with `krnr import set <file>`; imports handle name collisions by appending `-import-N` suffixes and DB imports support an `--overwrite` flag.
- Import: add `--on-conflict` policy to `krnr import set` and `krnr import db` (per-set policy) with values `rename` (default), `skip`, `overwrite`, and `merge`.
  - `merge` appends incoming commands to existing command sets and records a new version snapshot; use `--dedupe` to remove exact-duplicate commands when merging.
  - `skip` ignores imported sets when a name collision exists; `overwrite` replaces the existing set with the imported one; `rename` (default) appends `-import-N` to avoid collisions.
- Interactive CLI: running `krnr import` or `krnr export` with no arguments now starts an interactive prompt to select the type (db or set) and enter options (paths, conflict policy, dedupe), while still supporting the existing one-liner flags for non-interactive use.
- Tests: add unit and integration tests for `--on-conflict` policies and interactive import flows.
- Docs: update `docs/importer.md` and `docs/cli.md` with the new `--on-conflict` and `--dedupe` flags and interactive examples.

## v1.0.0 - 2026-01-11 (released)

- Major: v1.0.0 release candidate.
- Key features included in v1.0.0:
  - Tagging & Search UI: `krnr tag add|remove|list` and `--tag` filtering for `krnr list`.
  - Text search and fuzzy search: `--filter` (substring search) and `--fuzzy` (case-insensitive subsequence matching) implemented in `internal/registry`.
  - Parameters & Variable Substitution: `--param` flag with `{{param}}` syntax, interactive prompting, and `env:VAR` support implemented in `krnr run`.
  - Versioning & History: add `command_set_versions` snapshots, `krnr history <name>` and `krnr rollback <name> --version` CLI commands with tests and migration support.
  - Unit tests and integration tests for tags, search, fuzzy matching, parameters, and versioning (CLI + repository tests).
  - CLI docs updated (`docs/cli.md`) with examples for `--tag`, `--filter`, `--fuzzy`, `history`, and `rollback`.
  - Security & Safety hardening: conservative destructive command checks with `--force` override, parameter redaction for env-bound and secret-like params, `krnr delete` and `krnr install` guardrails and confirmations, `docs/security.md` added, and safety tests; CI SAST scans planned.
  - Packaging & Windows installer: Windows MSI (WiX) added with License dialog, Start Menu shortcuts, uninstall support, PATH handling, and ARP metadata (contact and links). The release workflow now builds and uploads the MSI to GitHub Releases on tag commits (see `.github/workflows/release.yml`).
  - Packaging manifests: winget, Scoop, and Chocolatey manifest templates added under `packaging/windows/` and `packaging/` (placeholders for release URLs and SHA values); CI automation to replace placeholders and open PRs is pending.
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
