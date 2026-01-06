# krnr — Kernel Runner

krnr is a cross-platform CLI tool that provides a global, persistent command registry backed by SQLite.

Quick start (dev):

1. Install Go 1.22+
2. Run `go build ./...`
3. Run `./krnr` (or `krnr.exe` on Windows)

Database:

- The database file is created under your home directory in `.krnr/krnr.db` by default.
- The schema and migrations are embedded and are applied automatically on first run.

Tests:

Run unit tests with:

```bash
# run all tests
go test ./...

# run database tests only
go test ./internal/db -v
```

See `PROJECT_OVERVIEW.md` and `docs/database.md` for design notes and schema.
