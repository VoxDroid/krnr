# krnr — Kernel Runner

krnr is a cross-platform CLI tool that provides a global, persistent command registry backed by SQLite.

Quick start (dev):

1. Install Go 1.22+
2. Run `go build ./...`
3. Run `./krnr` (or `krnr.exe` on Windows)

Database:

- The database file is created under your home directory in `.krnr/krnr.db` by default.
- You can override the data directory and DB path using environment variables:
  - `KRNR_HOME` — set the data directory
  - `KRNR_DB` — set the full path to the DB file (takes precedence)
- The schema and migrations are embedded and are applied automatically on first run.

Tests:

Run unit tests with:

```bash
# run all tests
go test ./...

# run database tests only
go test ./internal/db -v
```

See `PROJECT_OVERVIEW.md`, `docs/config.md` and `docs/database.md` for design notes and schema.

Linting & formatting

- Format the code with:

```bash
./scripts/fmt.sh
```

- Run linters with:

```bash
./scripts/lint.sh
```

Pre-commit hooks

Install pre-commit and enable the hooks:

```bash
pip install pre-commit
pre-commit install
```

This repository includes a `.pre-commit-config.yaml` that runs basic checks, `gofmt`/`goimports`, and `golangci-lint`.

