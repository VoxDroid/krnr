CREATE TABLE IF NOT EXISTS command_sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    last_run DATETIME
);

CREATE TABLE IF NOT EXISTS commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_set_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    command TEXT NOT NULL,
    FOREIGN KEY(command_set_id) REFERENCES command_sets(id)
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS command_set_tags (
    command_set_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (command_set_id, tag_id)
);

-- Versioning: snapshots of a command_set at change points.
CREATE TABLE IF NOT EXISTS command_set_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_set_id INTEGER NOT NULL,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    author_name TEXT,
    author_email TEXT,
    description TEXT,
    commands TEXT NOT NULL, -- JSON array of commands or newline-separated commands
    operation TEXT NOT NULL, -- e.g., 'create','update','delete','rollback'
    FOREIGN KEY(command_set_id) REFERENCES command_sets(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_command_set_versions_unique ON command_set_versions (command_set_id, version);
