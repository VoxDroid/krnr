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

See `PROJECT_OVERVIEW.md`, `docs/config.md`, `docs/database.md`, `docs/architecture.md`, and `docs/roadmap.md` for design notes, architecture, and roadmap.

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

Linting on Windows (notes)

On Windows you may encounter an error from `golangci-lint` related to incompatible export-data between Go toolchains ("could not load export data", "unsupported version"). If that happens, use the Docker fallback included in this repo:

```bash
# runs the linter inside a container (no local install required)
docker run --rm -v "$(pwd)":/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run --verbose
```

Continuous Integration

This repository includes a GitHub Actions workflow (`.github/workflows/ci.yml`) which runs on push and pull requests. It performs formatting, linting (uses the official golangci-lint action), unit tests, and produces cross-platform build artifacts for Linux, macOS, and Windows (amd64 and arm64). Artifacts are attached to the workflow run for download.

