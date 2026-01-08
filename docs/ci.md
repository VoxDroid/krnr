# CI and Linting

This document explains CI conventions and how the repository handles linting on
various environments.

## Linting

We use `golangci-lint` for static analysis. Because `golangci-lint` binaries can
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
