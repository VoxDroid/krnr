# Exporter

The exporter provides functions to export the active database or individual command sets to a portable SQLite file.

- `ExportDatabase(dstPath)` — copies the current database file to `dstPath`.
- `ExportCommandSet(srcDB, name, dstPath)` — writes a new SQLite database file containing only the named command set and its commands.

Usage examples are in `internal/exporter/export_test.go` and `internal/importer/import_test.go`.
