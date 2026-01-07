# Registry

The registry package (`internal/registry`) provides CRUD operations for `CommandSet` and `Command` models.

Key functions:

- `NewRepository(db *sql.DB) *Repository` — construct a repository.
- `(*Repository) CreateCommandSet(name string, description *string) (int64, error)` — create a new named workflow.
- `(*Repository) AddCommand(commandSetID int64, position int, cmd string) (int64, error)` — add a command to a set.
- `(*Repository) GetCommandSetByName(name string) (*CommandSet, error)` — fetch set and its ordered commands (includes `Tags`).
- `(*Repository) ListCommandSets() ([]CommandSet, error)` — list sets (without commands; includes `Tags`).
- `(*Repository) DeleteCommandSet(name string) error` — delete a set and its commands.
- `(*Repository) AddTagToCommandSet(commandSetID int64, tag string) error` — add a tag to a set (creates tag if needed).
- `(*Repository) RemoveTagFromCommandSet(commandSetID int64, tag string) error` — remove a tag from a set.
- `(*Repository) ListTagsForCommandSet(commandSetID int64) ([]string, error)` — list tags for a set.
- `(*Repository) ListCommandSetsByTag(tag string) ([]CommandSet, error)` — list sets that have the given tag.
- `(*Repository) SearchCommandSets(query string) ([]CommandSet, error)` — search sets by name, description, or contained commands.

Tags are stored in `tags` and `command_set_tags` tables; the `Get` and `List` calls populate the `Tags` field on returned `CommandSet` values.

The tests in `internal/registry/registry_test.go` perform basic end-to-end CRUD operations against an in-memory SQLite file created with the project's DB initialiser.
