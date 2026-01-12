# krnr v1.1.0 — UX and import/export improvements (2026-01-12)

This patch release improves CLI UX and introduces portable import/export support.

Highlights

- Interactive save behavior — `krnr save` prompts for a new name when a duplicate name is provided instead of failing.
- Export (`krnr export`) and import (`krnr import`) commands to transfer DBs and individual sets as portable SQLite files.
- Per-set conflict handling on import with `--on-conflict` (rename|skip|overwrite|merge) and `--dedupe` for merges.
- Interactive top-level `krnr export` and `krnr import` prompts for users who prefer guided flows.

For full details, see `CHANGELOG.md` and `docs/releases/v1.1.0.md`.
