# Architecture

krnr (Kernel Runner) is designed as a small, modular Go application with clear separation of concerns:

- CLI Layer (`cmd/`)
  - Built using Cobra, provides user-facing commands (`save`, `run`, `list`, `describe`, `edit`, `delete`, `export`, `import`).
  - Commands are thin: they parse flags and orchestrate calls into services.

- Registry Service (`internal/registry`)
  - Encapsulates all CRUD operations for `CommandSet` and `Command` models.
  - Uses SQLite as persistent backing store through `internal/db`.

- Database Layer (`internal/db`)
  - Initializes the data directory and opens the SQLite connection.
  - Applies embedded schema migrations at startup via `ApplyMigrations`.

- Execution Engine (`internal/executor`)
  - OS-aware command execution wrapper with `DryRun` and `Verbose` modes.
  - Streams stdout/stderr through caller-provided writers so the CLI can forward or capture output.

- Utilities (`internal/utils`)
  - Editor opener (`OpenEditor`) that respects `$EDITOR` and provides sensible fallbacks. The editor helper is testable by setting `EDITOR` to a script during tests.
  - Recorder (`internal/recorder`) — small helper to record commands from stdin and save them into the registry. Useful for interactive capture of multi-line workflows.
  - User identity (`internal/user`) — persistent, simple author identity (`whoami`) that can be used as default author metadata for `save` operations.
  - Confirmation helper for interactive flows.

- Importer & Exporter (`internal/importer`, `internal/exporter`)
  - Support exporting a single command set or the full DB into a portable SQLite file and importing them back with collision handling.

Design notes:
- Global-first: the DB is location-aware (see `internal/config`) and uses a single file (`~/.krnr/krnr.db` by default) to provide global access across directories and shells.
- Scriptless: workflows are stored in the database not in per-project script files.
- Safe defaults: commands do not run in background; default behavior is stop-on-error.
- Cross-platform: execution paths use OS-native shells (`cmd/pwsh` on Windows, `bash` on Unix).

Diagram (conceptual):

```
CLI (cobra)
   └─> Registry (CRUD) ↔ DB (SQLite)
   └─> Executor (OS-aware) → Shell
   └─> Importer/Exporter ↔ Files
```

Operational notes:
- Unit tests are located under `internal/*` for each package. Integration tests exercise realistic save→run→export→import flows.
- CI runs formatters, linters and tests; GitHub Actions builds cross-platform artifacts.

For more details see `PROJECT_OVERVIEW.md` and `docs/database.md` and `docs/cli.md`.
