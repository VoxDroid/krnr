# Roadmap

This roadmap outlines short-, medium-, and long-term priorities for `krnr`.

## Short-term (v0.x)
- Complete unit and integration tests for all core packages (`db`, `registry`, `executor`, `cmd`).
- Add `export` and `import` CLI commands and document usage. (Exporter/Importer implemented in code and tests.)
- Improve documentation (architecture, database, CLI) and onboarding notes (this is in progress).
- Add pre-commit hooks and CI (format, lint, tests) — done.

## Medium-term (v1.0)
- Add advanced features:
  - Tagging and search operations UI (`tags`, filters, fuzzy search)
  - Parameters & variable substitution for commands
  - Versioning/history of command sets
- Packaging & releases: cross-platform binaries, Homebrew / Scoop manifests
- Security review & safety hardening (prevent accidental destructive operations)

## Long-term (v1.x+)
- Collaboration & sharing: remote registries, publishable command sets
- UI integrations: TUI / VSCode extension to browse and run workflows
- Telemetry & metrics (opt-in) for usage analysis

## Milestones
- v0.1: Core features (save, run, list, describe, edit, delete) + DB + basic tests
- v0.5: Import/Export, cross-platform builds, comprehensive tests
- v1.0: Stability, full docs, packaging, security checklist complete

Contributions welcome — see `CONTRIBUTING.md` for guidelines (TODO).
