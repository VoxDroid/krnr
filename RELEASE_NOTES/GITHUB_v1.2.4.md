# v1.2.4 - 2026-02-04

- Bugfix: Prevent UI deformation when running commands that emit terminal control sequences (alternate-screen, clear-screen, cursor movements, OSC sequences). The TUI now sanitizes run output and preserves SGR color codes while removing destructive sequences.
- Tests: Add `TestRunStreamsSanitizesControlSequences` and sanitizer unit tests to prevent regressions.
- Quality: Addressed gocyclo and golangci-lint items; no remaining high-complexity functions and no linter errors.

Upgrade notes: This is a backwards-compatible UI fix; no DB or CLI changes.
