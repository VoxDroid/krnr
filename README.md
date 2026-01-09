# krnr — Kernel Runner

krnr is a cross-platform CLI tool that provides a global, persistent command registry backed by SQLite.

Quick start (dev):

1. Install Go 1.22+
2. Run `go build ./...`
3. Run `./krnr` (or `krnr.exe` on Windows)

Install & setup (users):

- Download a release binary from the GitHub Releases page or build a local binary and use the included installer to place it on your PATH:
  - `./krnr install --user --from ./krnr --add-to-path` (Unix/macOS)
  - `.\krnr.exe install --user --from .\krnr.exe --add-to-path` (PowerShell/Windows)
- Verify with `krnr status` and start a new shell session if necessary.
- See `docs/install.md` for full installation guidance, PATH handling details, and troubleshooting tips.


Clean rebuild (dev):

- If you see behavior that looks out of date (for example, after fixing runtime output), perform a clean rebuild to ensure you're running the latest code you have checked out:

```bash
# Unix / macOS
go clean -cache -testcache
go build -v -o krnr .
./krnr run <name>

# Windows (PowerShell)
go clean -cache -testcache
go build -v -o krnr.exe .
.\krnr.exe run <name>
```

- For release packaging and cross-compiles use the provided scripts:
  - `./scripts/build.sh` — cross-compile binaries for supported platforms
  - `./scripts/release.sh <version>` — create release archives (tar/zip) locally

- If you built from a previously released artifact (`dist/krnr-...`), replace it with the freshly-built binary to pick up fixes.
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

This repository includes a GitHub Actions workflow (`.github/workflows/ci.yml`) which runs on push and pull requests. It performs formatting, linting (with a local or Docker fallback), unit tests, and produces cross-platform build artifacts for Linux, macOS, and Windows (amd64 and arm64). Artifacts are attached to the workflow run for download.

Release packaging

A release workflow (`.github/workflows/release.yml`) will run when you push a tag like `v1.2.3`. It builds platform binaries, packages them into `.tar.gz` / `.zip` archives, generates a SHA256 sums file, and attaches the artifacts to a GitHub Release.

To create a local release build (without the workflow), run:

```bash
# Option 1: provide version explicitly
./scripts/release.sh v0.1.0

# Option 2: bump the root VERSION file and run the script
# edit VERSION and then:
./scripts/release.sh
```

This will create archives under `dist/` which embed the version (from `VERSION` or the provided arg).
