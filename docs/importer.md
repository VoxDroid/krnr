# Importer

The importer supports importing entire SQLite databases or individual command sets exported by `krnr export`.

CLI usage

- Import a single exported set: `krnr import set path/to/file.db [--on-conflict=rename|skip|overwrite|merge] [--dedupe]`
  - `--on-conflict` controls how name collisions are handled:
    - `rename` (default): append a `-import-N` suffix to conflicting names (non-destructive).
    - `skip`: do not import a set if a command set with the same name already exists.
    - `overwrite`: delete the existing set and insert the imported set.
    - `merge`: append incoming commands to the existing set; use `--dedupe` to remove exact-duplicate commands when merging.
- Import an entire DB: `krnr import db path/to/full.db --overwrite` — overwrites the active database file only when `--overwrite` is specified. Alternatively, use `krnr import db path/to/full.db --on-conflict=merge --dedupe` to apply the per-set conflict policy when merging sets into an existing DB.

- `ImportDatabase(srcPath, overwrite bool, opts ImportOptions)` — copy a DB file into the active data path or apply per-set conflict policies when the destination exists.
- `ImportCommandSet(srcPath, opts ImportOptions)` — import all command sets found in `srcPath` into the active DB; use `opts.OnConflict` and `opts.Dedupe` to control behavior.

Interactive mode

- Running `krnr import` with no arguments will start an interactive prompt to select `db` or `set`, ask for the source file, and walk through conflict policy and dedupe options. This is useful for interactive workflows while one-liners are still supported for scripting.
