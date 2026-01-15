# TUI Modularization Plan

## Goals ✅
- Break up `cmd/tui/ui/tui.go` into smaller, well-scoped modules to improve readability, maintainability, and testability.
- Make it easier to reason about input handling, rendering, and business logic by separating concerns.
- Preserve behavior via tests and add new unit/behavioral tests for each module.
- Minimize user-visible regressions by using incremental, reversible changes and continuous testing.

---

## High-level module boundaries (proposed) 🔧
1. core (cmd/tui/ui/core.go)
   - Bubble Tea model definition (`TuiModel` state fields), Init and main `Update`/`View` wiring.
   - Small, high-level helper functions that orchestrate sub-modules.
   - Only coordinates other modules; keeps logic minimal.
2. input (cmd/tui/ui/input.go)
   - All key/message handling split into focused functions: editor input, list navigation, versions navigation, modal handling, and top-level message dispatching helpers.
   - Expose small, pure-ish helpers for interpreting KeyMsg -> action (useful for unit tests).
3. editor (cmd/tui/ui/editor.go)
   - All editor state and editing helpers, including keystroke handling for editor fields, editing commands, add/delete, save/cancel, and in-editor sanitization hooks.
4. rendering (cmd/tui/ui/render.go)
   - All functions that compose the final `View()` output: layout helpers, header/footer rendering, detail formatting helpers (`formatCSFullScreen`, `formatCSDetails`, `formatVersionDetails`), and styles.
5. versions (cmd/tui/ui/versions.go)
   - Version-list specific logic: building `versionsList`, preview content management, `setVersionsPreviewIndex`, rollback prompts and helpers.
6. viewport (cmd/tui/ui/viewport.go)
   - Helpers for creating and updating `viewport.Model`, preserving offsets (Y), scroll-to functions, and scroll indicators.
7. tests (cmd/tui/ui/*_test.go)
   - Keep tests colocated and add unit tests for each module (input, editor behaviors, versions preview logic, render snapshots for some pieces where helpful).
8. adapters/interfaces (cmd/tui/ui/interfaces.go)
   - Define small interfaces for external dependencies that the TUI uses (registry, executor adapters) so modules can be tested in isolation with fakes.

---

## Design principles & constraints 💡
- Keep each file under ~300 lines when reasonable (prefer <200 lines).
- Public package API should remain limited; prefer internal helpers unexported unless needed by tests.
- Preserve `TuiModel` as the single source of truth for UI state.
- Favor dependency injection for testability (use adapters and fake implementations in tests).
- Maintain backward compatibility: behavior and CLI interactions must not change.
- Tests are required for any moved/modified logic before approval.

---

## Detailed checklist (migration steps) 🧭
Phase 0 — Prep
- [ ] Add `krnr_docs/TUI_modularization.md` (this file) and commit as a plan.
- [ ] Create small smoke tests that exercise the main flows so we can detect regressions quickly (e.g., open detail, edit/save commands, versions preview update, run with edited commands).
- [ ] Add interfaces/go doc comments for public helpers to clarify responsibilities.

Phase 1 — Split rendering (IN PROGRESS)
- [x] Create `render.go` and move `formatCSFullScreen`, `formatCSDetails`, `formatVersionDetails`, and other rendering helpers into it.
- [x] Add unit tests for the formatting functions (string output asserts for representative inputs).
- [x] Run `go test ./cmd/tui/ui -run Test* -v` and fix regressions.
- [ ] Status: Phase 1 extraction complete; follow-up: review and refine exported helpers and add more rendering snapshot tests if needed.

Phase 2 — Split versions & viewport (COMPLETED)
- [x] Create `versions.go` and move `setVersionsPreviewIndex`, `renderVersions`, `formatVersionPreview`, and `formatVersionDetails` there.
- [x] Create `viewport.go` and add `ensureViewportSize` helper used by `View()`.
- [x] Add tests: `TestSetVersionsPreviewIndexUpdatesContentAndResetsOffset` verifies preview updates and offset reset.
- [x] Run `go test ./cmd/tui/ui -run TestVersions* -v` and fix regressions.
- [ ] Follow-up: add more viewport unit tests and a PTY-based integration test for versions preview if desired.

Phase 3 — Extract editor logic (COMPLETED)
- [x] Create `editor.go`. Editor state and keystroke handlers (add/delete/save) were moved into `cmd/tui/ui/editor.go` and delegate to `executor.Sanitize/ValidateCommand` via adapters as needed.
- [x] Add targeted editor tests: typing behavior (including `k`, `j`, space), sanitization log/display, save rejection on invalid commands (tests added in `editor_test.go`).
- [x] Run `go test ./cmd/tui/ui -run TestEditor* -v` and fixed regressions (package now passes tests).
- [ ] Follow-up: add more editor-focused unit tests (field navigation, logging visibility, edge cases for command sanitization).

Phase 4 — Input & dispatch (COMPLETED)
- [x] Create `input.go` scaffold and initial test file (`input_test.go`) with basic test placeholders.
- [x] Move filter-mode handling into `input.go` (done).
- [x] Move most key handling helpers from `tui.go` into `input.go` (dispatchKey, handleGlobalKeys, handleFocusedNavigation, confirm flows).
- [x] Add unit tests for global interpreter functions and run behavior (e.g., quit/help/tab/run and headless run streaming) in `input_test.go`.
- [x] Move built-in list filtering handling into `input.go` (added `handleListFiltering`/`applyListFilterKey`) and added tests.
- [x] Finalize `Update()` delegation so it acts as a thin orchestrator and defers input handling to helpers.
- [ ] Add additional unit tests for edge cases (focus toggles, Enter-on-versions behavior, confirm flow restores, deep-scroll/version preview interactions).
- [ ] Add PTY-based integration tests to validate end-to-end edit→save→run flow (sanitization and no CreateProcess errors).

Phase 5 — Core & interfaces (COMPLETED)
- [x] Create `core.go` that contains the orchestration helpers: `NewModel`, `NewProgram`, and `Init` (now delegates to module helpers in other files).
- [x] Add `interfaces.go` to declare local `UIModel` interface that the TUI depends on (enables passing fakes and decouples UI from `modelpkg.UIModel`).
- [x] Moved small utilities and helpers (`readLoop`, `uniqueDestPath`, `trimLastRune`, `filterEmptyLines`) into `core.go` to centralize orchestration utilities.
- [x] Re-run the entire `cmd/tui/ui` test package and resolve issues (tests pass locally).
- [x] Follow-up: consider extracting additional small interfaces if needed as Phase 6 approaches.

Phase 6 — Tests & CI
- [ ] Expand headless tests to cover edge cases (terminal narrow, command sequences with sanitized characters, paging with many versions).
- [ ] Add one PTY-based integration test that runs the TUI and performs a realistic user edit/save/run flow and asserts no CreateProcess errors.
- [ ] Ensure tests run under CI and Windows (Windows skips for shell-specific tests remain allowed).

Phase 7 — Clean up & docs
- [ ] Run `golangci-lint` and address warnings that are meaningful.
- [ ] Add package-level comments in `cmd/tui/ui` describing the new file separation and design rationale.
- [ ] Update `krnr_docs/TUI_modularization.md` with any lessons learned and the final layout.

---

## Testing matrix & acceptance criteria ✅
- Unit tests passing for `cmd/tui/ui` package (no regressions), especially editor, versions, and render tests.
- `go test ./...` returns exit code 0 in CI (Windows and Linux matrix in the CI pipeline); any OS-specific failures must include a skip reason.
- PTY/E2E: basic scenario that replicates the real user workflow (open editor -> type smart quotes & NUL -> save -> run set) runs without CreateProcess "invalid argument" errors.
- Visual verification: run `./krnr tui` and exercise common flows manually. Keep a short checklist of manual checks for reviewer.

---

## Rollout & review process 📋
1. Do the migration one phase at a time, opening a small PR per phase.
2. Each PR must include:
   - Tests that cover moved logic (unit and/or integration as applicable).
   - A short description in the PR about why code was moved and what was removed/added.
   - Screenshots (optional) or a short GIF for UI behavior changes if any.
3. Merge after CI passes and one reviewer approves.
4. After all phases, run a smoke `e2e` check and tag release notes where appropriate.

---

## Risks & mitigations ⚠️
- Risk: small changes accidentally alter visual output. Mitigation: use string-based unit tests for formatting and a headless TUI test harness to detect changes.
- Risk: test flakiness in PTY tests (timing issues). Mitigation: keep PTY tests minimal and robust to timing by waiting for stable output markers.
- Risk: import cycles during refactor. Mitigation: extract small interfaces (`interfaces.go`) early and use them to decouple modules.

---

## Next steps (short-term) ⏭️
- Start with Phase 1: extract rendering helpers into `render.go`, add tests, and open a small PR.
- After the PR lands, proceed to Phase 2 (versions + viewport) and iterate.

---

If you'd like, I can open the first PR that extracts `render.go` and include a set of tests and description ready for review. Which phase should I start with? (Recommended: Phase 1 — extract rendering.)
