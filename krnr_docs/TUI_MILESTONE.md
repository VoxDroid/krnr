# TUI Milestone

**Status:** In progress (prototype + initial features implemented)

Date: 2026-01-12

---

## Summary ✅

This document tracks the TUI (Terminal UI) milestone for `krnr`. The TUI is a keyboard-first, accessible terminal interface that provides interactive equivalents to common CLI workflows while reusing the existing core packages (`registry`, `executor`, `importer`, `exporter`).

The purpose of this milestone is to ship a well-tested, documented and supported `krnr tui` flow that covers the main interactive use cases (browse, describe, run with params, view logs, import/export, edit/save, rollback, tags, install/uninstall).

---

## What is implemented (current PoC / ✅)

- Prototype Bubble Tea implementation under `cmd/tui` (PoC: list + details). ✅
- Deterministic two-column preview renderer (labels & wrapped values), fixes preview alignment and ensures commands display consistently. ✅
- Selection-change preview updates (fetches full CommandSet and updates the right-hand pane immediately). ✅
- `Init()` sizing/population fixes so content appears on first launch without a resize/scroll. ✅
- Run streaming support (start run, stream events, viewport tailing) and associated headless handling. ✅
- Headless UI tests (unit-level) validating layout, alignment, and behavior. ✅
- PTY-based E2E test added (skips on unsupported platforms). ✅
- Framework-agnostic UI model in `internal/tui` with adapters for registry and executor (keeps UI thin and testable). ✅
- `formatCSDetails` and `formatCSFullScreen` produce an "invisible border" table layout for preview/detail. ✅

---

## TUI <-> CLI feature mapping (what users can do interactively)

The target scope of `krnr tui` is parity for interactive workflows that the CLI exposes. The list below shows which CLI commands currently have working interactive equivalents in the TUI (checked) and which are still planned (unchecked):

- [x] `krnr list` — browse list, filter, fuzzy search
- [x] `krnr describe <name>` — preview pane & full-screen details
- [x] `krnr run <name>` — run (basic run + streaming logs) and dry-run preview (parameter editor modal planned)
- [x] `krnr save` / `krnr edit` — interactive save/create/edit flows (modal UI implemented; press `c` to create, `e` to edit)
- [x] `krnr record` — record commands interactively (future modal)
- [x] `krnr import` / `krnr export` — import/export flow (interactive helpers) (menu modal added; export/import DB and sets available via `m` Menu)
- [x] `krnr install` / `krnr uninstall` — installer actions available from `m` Menu
- [x] `krnr whoami` — accessible from `m` Menu (shows stored identity)
- [x] `krnr history <name>` & `krnr rollback` — view history, rollback UI
- [ ] `krnr tag` add/remove/list — tag management UI
- [ ] `krnr install` / `krnr uninstall` — installer views and dry-run planning
- [ ] `krnr whoami` — identity management for saves
- [ ] `krnr status` — diagnostics

---

## Outstanding todos & next steps (📝)

- ✅ Implemented interactive Create/Save modal (press `c` to create; form collects name, description, commands, author, email, tags). Unit tests added to verify save flow.
- Implement parameter-editor modal for `run` (name/value with `env:VAR` resolution and secret redaction).
- Add a command-by-command dry-run viewer with redaction support in full-screen detail.
- Visual polish: optional full-line background blocks for command rows and color/theme refinements (maintain testability). 
- Accessibility: keyboard help modal, high-contrast theme tuning, screen reader testing and aria-like support (where applicable).
- Expand PTY E2E coverage and add CI jobs to run headless/PTY tests on Linux/macOS runners (ensure Windows behavior is covered via skipping assertions or separate expectations).
- Add interactive flow tests (headless) for create/save/import/export and rollback.
- Add usage docs & screenshots (`docs/cli.md` TUI section, `docs/releases/v1.2.0.md`, a TUI guide page with keyboard reference).
- Packaging/release: ensure `krnr tui` is included in release artifacts and release notes.

---

## Acceptance criteria

- `krnr tui` is documented and discoverable via `--help` / `docs/cli.md`.
- Users can perform common interactive flows via the TUI with equivalent safety and confirmations as the CLI.
- The TUI code uses `internal/tui` adapters and reuses core logic from the CLI packages (no duplicated business logic).
- Headless UI tests and PTY E2E tests cover interaction and rendering; CI jobs run them on supported platforms.

---

## How to test

- Unit & headless tests: `go test ./cmd/tui/... -v`
- PTY E2E (platform-specific): run `go test ./cmd/tui/... -run TestTuiInitialRender_Pty -v` in a PTY-capable environment (Linux/macOS) or use a terminal multiplexer on CI.

---

## Files & references

- Implementation: `cmd/tui` (Bubble Tea model), `internal/tui` (UIModel & adapters)
- Tests: `cmd/tui/ui/tui_test.go` (headless), `cmd/tui/ui/pty_e2e_test.go` (PTY E2E)
- Docs: `docs/cli.md` (TUI section), `docs/releases/v1.2.0.md`

---

## Notes / Next owner items

- I will implement the Create/Save modal or visual polish next depending on your priority — tell me whether you prefer color/styling parity first or the interactive create flow. 

---
