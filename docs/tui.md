# TUI Design (draft)

This document describes the design and architecture for the `krnr` Terminal UI (TUI).
It is a living document and will be refined as we iterate on the UX and implementation.

Goals
- Provide interactive equivalents for the existing CLI commands so users can perform the same interactive tasks through visuals (parity for interactive workflows).
- Keep the CLI as the canonical scripting interface; the TUI delegates to internal packages and acts as a presentation layer.
- Keyboard-first, accessible, testable architecture with headless tests and cross-platform CI coverage.

High-level architecture

- `cmd/tui` — program entrypoint. Wires the UI framework (Bubble Tea) to an `internal/tui` application model and adapters.
- `internal/tui` — framework-agnostic domain layer: adapter interfaces (Registry, Executor, ImporterExporter, Installer), UI models, and test helpers. This package must not import Bubble Tea or lipgloss.
- `internal/tui/adapters` — concrete adapters that implement the interfaces by delegating to existing packages (e.g., `registry`, `executor`, `exporter`, `importer`, `install`). These adapters live in `internal/tui/adapters` and are tested against `internal/tui` mocks.
- `cmd/tui/*` — Bubble Tea view & controller code that calls into `internal/tui` model and adapters.

Adapters & interfaces (summary)
- RegistryAdapter:
  - ListCommandSets() ([]registry.CommandSet, error)
  - GetCommandSetByName(name string) (registry.CommandSet, error)
  - SaveCommandSet(cs registry.CommandSet) error
  - DeleteCommandSet(name string) error
  - Tag operations, version listing & apply
- ExecutorAdapter:
  - RunCommandSet(ctx, name, opts) (RunHandle, error)
  - RunHandle exposes an events channel for stdout/stderr lines and a Cancel() function
- Importer/Exporter:
  - ExportSet(name, dst) error
  - ImportSet(file, policy) error
- InstallerAdapter:
  - Install(...)
  - Uninstall(...)

UI Model responsibilities (framework-agnostic)
- Load data for screens (list, details, history)
- Provide actions that return channels or commands (run + stream, import/export)
- Communicate progress and status via typed events, not UI framework primitives

Testing strategy
- Unit tests for `internal/tui` model using mock adapters.
- Headless UI tests: run Bubble Tea programs in a pseudo-tty (pty) in CI to assert rendering and interaction flows. We include simple headless unit tests that simulate key presses and run streaming behavior by using fake adapters; run them locally with `go test ./cmd/tui/... -v`.
- E2E harness to exercise run/streaming behavior with a fake executor runner.

Accessibility & theming
- High-contrast mode: toggle with `T` inside the TUI. This switches to a high-contrast color scheme to improve readability in low-vision or busy terminal themes.
- Menu modal: press `m` to open the **Menu** modal which contains actions like `Export database`, `Import database`, `Import set`, `Install`, `Uninstall`, and `Status`. Export/Import invoke the existing adapter-backed logic so the TUI delegates to the same exporter/importer paths as the CLI.

Key handling and spaces
- Note: different terminals may report the spacebar as either a `KeyRunes` event with a single `' '` rune or as a `KeySpace` event. The TUI now handles both forms consistently for editor input and the list filter so pressing the spacebar reliably inserts a space character regardless of environment. Tests were added to prevent regressions.

Sanitizing run output
- The TUI now sanitizes streaming command output shown inside the output viewport to remove control sequences that affect global terminal state (e.g., alternate screen, clear-screen, cursor movements, and OSC sequences) while preserving SGR color codes. Cursor-forward sequences are converted to spaces and cursor-horizontal-absolute to separators for readability. This prevents external commands like `fastfetch` from deforming the TUI layout. The sanitizer is conservative and tested; see `internal/tui/sanitize` for details.

Interactive commands & hybrid PTY
- The TUI supports running interactive commands that require user input (e.g., `sudo` password prompts, `pacman` confirmations). When a run is in progress, typed keys are forwarded to the process stdin.
- The executor uses a **hybrid PTY** approach: stdin and the controlling terminal use a PTY so programs that read from `/dev/tty` work, while stdout/stderr remain as pipes for viewport-friendly output.
- All prompts and output appear inside the **run output panel** (viewport), not in the footer or bottom bar.
- Output streams live — no keypress required to see results.


CI
- We added a GitHub Actions workflow `.github/workflows/tui-ci.yml` to run `go test ./... -v` across Ubuntu, Windows and macOS. The workflow validates headless UI tests and the rest of the test suite on PRs and pushes to `main`.
Accessibility
- Keyboard-first navigation and clear help (`?`), discoverable keybindings
- High-contrast theme option and avoid relying on color for meaning
- Add an accessibility checklist and include it in CI gates

Milestones
1. Adapter interfaces & `internal/tui` model skeleton
2. Adapters that bridge to `internal/registry` and `internal/executor` (mocked implementations for tests)
3. Bubble Tea UI components (list, detail, run modal, logs) and integration tests
4. Headless UI tests & CI integration
5. Accessibility, theming, docs, packaging changes

Notes
- Keep adapters small and easily mockable; prefer returning channels of typed events to tight-coupling the UI.
- Avoid UI logic in adapters; adapters should translate domain operations into simple primitives (list, get, run, stream, cancel).
