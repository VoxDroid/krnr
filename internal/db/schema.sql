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
