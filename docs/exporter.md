# Exporter

The exporter provides functions to export the active database or individual command sets to a portable SQLite file.

- `ExportDatabase(dstPath)` — copies the current database file to `dstPath`.
- `ExportCommandSet(srcDB, name, dstPath)` — writes a new SQLite database file containing only the named command set and its commands.

CLI usage

- Export whole DB: `krnr export db --dst path/to/file.db` (if `--dst` is omitted a default `./krnr-YYYY-MM-DD.db` is written; if that file exists a numeric suffix is appended to avoid overwrite)
- Export a single set: `krnr export set <name> --dst path/to/file.db`

Notes & Tips

- The export routines perform a WAL checkpoint where possible to ensure a consistent copy of the DB file is produced.
- Exporting a set writes a minimal DB containing only the set and its commands; this file is suitable for `krnr import set <file>` or manual shipping.

Usage examples are in `internal/exporter/export_test.go` and `internal/importer/import_test.go`.
