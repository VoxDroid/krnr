# v1.2.0 — Initial TUI release

This release introduces the initial Terminal UI (`krnr tui`) as an interactive entrypoint that complements the CLI without replacing it. The TUI delegates to existing internal packages (`registry`, `executor`, `importer`, `exporter`) so automation and scripting continue to use the CLI while users can interactively browse and run workflows.

Highlights

- `krnr tui` (v1.2.0) — initial release: browse, preview, full-screen detail, run with parameter prompting and streaming logs, save/edit flows, import/export helpers, history viewing and rollback, installer views, and `status` diagnostics.
- Tests: headless UI unit tests and a PTY E2E test were added.
- Docs: `docs/cli.md`, `docs/releases/v1.2.0.md`, and `krnr_docs/TUI_MILESTONE.md` updated to document usage and testing guidance.

See `CHANGELOG.md` for full details.
