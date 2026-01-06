# Importer

The importer supports importing entire SQLite databases or individual command sets exported
from other krnr databases.

- `ImportDatabase(srcPath, overwrite bool)` — copy a DB file into the active data path. Use with care; set `overwrite=true` to replace an existing DB.
- `ImportCommandSet(srcPath)` — import all command sets found in `srcPath` into the active DB; name collisions are handled by appending a suffix.
