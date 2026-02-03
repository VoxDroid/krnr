# v1.2.3 - 2026-02-03

- Bugfix: Fix spacebar behavior in the TUI. Some terminals produce a `KeySpace` event which previously was ignored; the TUI now treats it as a typed space in both the editor and list filter, restoring expected typing behavior.
- Tests: Add headless UI tests to cover `KeySpace` in filter and editor flows.
- Quality: Ran `gocyclo` and `golangci-lint` — no actionable issues found.

Upgrade note: This is a backward-compatible TUI fix; no DB or CLI-breaking changes.
