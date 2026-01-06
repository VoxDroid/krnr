# Registry

The registry package (`internal/registry`) provides CRUD operations for `CommandSet` and `Command` models.

Key functions:

- `NewRepository(db *sql.DB) *Repository` — construct a repository.
- `(*Repository) CreateCommandSet(name string, description *string) (int64, error)` — create a new named workflow.
- `(*Repository) AddCommand(commandSetID int64, position int, cmd string) (int64, error)` — add a command to a set.
- `(*Repository) GetCommandSetByName(name string) (*CommandSet, error)` — fetch set and its ordered commands.
- `(*Repository) ListCommandSets() ([]CommandSet, error)` — list sets (without commands).
- `(*Repository) DeleteCommandSet(name string) error` — delete a set and its commands.

The tests in `internal/registry/registry_test.go` perform basic end-to-end CRUD operations against an in-memory SQLite file created with the project's DB initialiser.
