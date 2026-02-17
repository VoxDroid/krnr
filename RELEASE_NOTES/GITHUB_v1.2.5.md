# v1.2.5 - 2026-02-17

This patch delivers **full interactive command support** in the TUI and fixes several output rendering issues.

## What's new

### Hybrid PTY for interactive commands
The TUI now uses a hybrid PTY approach: stdin and the controlling terminal use a PTY (so `sudo`, `pacman`, and other programs that read from `/dev/tty` work), while stdout/stderr remain as pipes (so programs like `fastfetch` produce clean, viewport-friendly output). Prompts appear **inside the run output panel**, not in the bottom bar.

### Live output streaming
Command output now streams live into the viewport immediately — no keypress required to see results.

## Bug fixes
- Fixed doubled output when stdout and stderr are the same writer in PTY mode
- Fixed UI border corruption from escape sequences split across read boundaries
- Fixed `nil pointer dereference` panic when PTY starter returned nil buffers
- Fixed truncated output for `fastfetch` and similar programs (buffer size 256→4096, EOF flush)
- Improved CSI sanitizer: cursor-forward → spaces, cursor-horizontal-absolute → separator, SGR colors preserved
- Fixed command list appearing empty on launch until a resize or keypress (`Init()` now returns a proper message for `Update()` to process)
- Fixed run output not visible when command has version history (versions-panel rendering was overwriting the viewport)
- Fixed inability to scroll run output after a run completes (auto-scroll now only active during run)
- Enter on versions panel now loads the selected version's detail into the left preview pane

## Quality
- `gocyclo` (>10) added to pre-commit; complex functions refactored
- All `golangci-lint` and `gocyclo` checks pass
- New tests: escape splitting, EOF flush, fd-reader, PTY simulation, sanitizer, headless viewport prompt

## New features
- **Delete version (D):** Press `D` on a highlighted version in the versions panel to delete that specific version record (with y/N confirmation)
- **Enter on versions:** Press Enter to view the full version detail in the left preview pane

## Upgrade notes
No DB schema changes. Simply upgrade to `v1.2.5` and interactive commands in the TUI will work automatically when running in a real terminal.
