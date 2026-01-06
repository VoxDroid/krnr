# Configuration & Paths

krnr stores data in a dot-directory by default and exposes a few environment
variables to customize behavior.

Defaults:
- Data directory: `~/.krnr` (Windows, Linux, macOS)
- Database file: `~/.krnr/krnr.db`

Environment overrides:
- `KRNR_HOME` — set the data directory to an explicit path
- `KRNR_DB` — set the full path to the SQLite DB file (overrides `KRNR_HOME`)

Functions:
- `internal/config.DataDir()` — returns the resolved data directory
- `internal/config.EnsureDataDir()` — creates the data directory if it doesn't exist
- `internal/config.DBPath()` — returns the resolved DB path

Notes:
- Use `KRNR_DB` if you want to place the DB on a specific drive or network share.
- `EnsureDataDir()` is called by the DB initializer to create the directory as needed.
