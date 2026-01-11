# CI and Linting

This document explains CI conventions and how the repository handles linting on
various environments.

## Linting

We use `golangci-lint` (v2.8.0) for static analysis. Because `golangci-lint` binaries can
be built against a different Go toolchain and produce `export-data` errors (e.g.
"unsupported version" or "could not load export data"), the repository provides
a resilient `scripts/lint.sh` script that:

- Attempts to run a locally-installed `golangci-lint` first.
- If missing, or if the local invocation fails with an export-data incompatibility,
  it attempts a Docker-based fallback using the official `golangci/golangci-lint`
  image.
- If Docker isn't available, the script prints actionable guidance on how to fix
  the environment and exits appropriately.

The CI workflow runs the formatter and the linter on Unix runners and uses the
Docker fallback in environments where the local linter is unavailable or
incompatible.

## Cyclomatic complexity (gocyclo)

We run `gocyclo` as part of `golangci-lint` to keep cyclomatic complexity in
check. The project warns on functions with cyclomatic complexity > 15 and
maintainers should aim to keep functions small and testable.

- If `gocyclo` flags a function (e.g., `gocyclo` reports complexity > 15), prefer
  extracting helper functions and adding focused unit tests, rather than writing
  long monolithic functions. Recent work reduced complexity for several functions
  (`internal/install.Uninstall`, `internal/install.addToPath`,
  `internal/install.GetStatus`, `internal/recorder.RecordCommands`, and
  `internal/importer.ImportCommandSet`).
- To run locally: `golangci-lint run --enable gocyclo` (or `./scripts/lint.sh`).
