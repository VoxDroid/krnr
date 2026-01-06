# Database (SQLite)

This project uses a single, embedded SQLite database to store global command sets and commands.

## Location

- Windows: `C:\Users\<user>\.krnr\krnr.db`
- Linux/macOS: `~/.krnr/krnr.db`

The path is resolved by `internal/config/paths.go`.

## Schema

See `internal/db/schema.sql` for the canonical schema. Key tables:

- `command_sets` — metadata about named workflows
- `commands` — ordered commands within a command set
- `tags` and `command_set_tags` — tagging support

## Migrations

Migrations are applied at startup by `internal/db.ApplyMigrations` (the schema is embedded).

To run tests that exercise DB creation and migrations:

```bash
go test ./internal/db -v
```
