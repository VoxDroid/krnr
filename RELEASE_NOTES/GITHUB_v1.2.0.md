# v1.2.0 — TUI initiative (planned)

Planned long-term release focusing on TUI integration across the application.

This release will not remove or change the existing CLI; instead, the TUI will act as a supported visual layer that calls into the existing internal packages (`registry`, `executor`, `importer`, `exporter`) so users who prefer a visual UI can perform common tasks while automation and scripting continue to use the CLI.

Scope: The TUI aims to provide interactive equivalents for all existing CLI commands (save, edit, run with parameter entry and streaming logs, list, describe, history, rollback, import/export, tag management, install/uninstall/status), while leaving the CLI unchanged as the authoritative scripting interface.

See `CHANGELOG.md` (v1.2.0 planned) and `docs/releases/v1.2.0.md` for implementation goals and milestones.
