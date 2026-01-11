# Roadmap

This roadmap outlines short-, medium-, and long-term priorities for `krnr`.

## Short-term (v0.x) — Completed
- Complete unit and integration tests for all core packages (`db`, `registry`, `executor`, `cmd`) — done.
- Add `export` and `import` CLI commands and document usage — done.
- Improve documentation (architecture, database, CLI) and onboarding notes — done.
- Add pre-commit hooks and CI (format, lint, tests) — done.

## Medium-term (v1.0)
- Add advanced features:
  - Tagging and search operations UI (`tags`, filters, fuzzy search) — **Implemented** (CLI + registry helpers + tests)
  - Parameters & variable substitution for commands
  - Versioning/history of command sets
- Security review & safety hardening (prevent accidental destructive operations) — **Completed** (conservative checks, redaction, prompts/tests; CI SAST planned)

## Long-term (v1.x+)
- Collaboration & sharing: remote registries, publishable command sets
- UI integrations: TUI / VSCode extension to browse and run workflows
- Telemetry & metrics (opt-in) for usage analysis

## Milestones
- v0.1: Core features (save, run, list, describe, edit, delete) + DB + basic tests
- v0.5: Import/Export, cross-platform builds, comprehensive tests
- v1.0: Stability, full docs, packaging, security checklist complete — see `docs/releases/v1.0.0.md` for release notes

Contributions welcome — see `CONTRIBUTING.md` for guidelines (TODO).
