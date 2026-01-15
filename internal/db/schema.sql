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

-- Ensure names are non-empty (trimmed) on insert and update. Use triggers so
-- existing databases will receive this protection when migrations run.
CREATE TRIGGER IF NOT EXISTS command_sets_check_name_insert
BEFORE INSERT ON command_sets
FOR EACH ROW
WHEN trim(NEW.name) = '' OR typeof(NEW.name) != 'text'
BEGIN
    SELECT RAISE(ABORT, 'invalid name: name cannot be empty or non-text');
END;

CREATE TRIGGER IF NOT EXISTS command_sets_check_name_update
BEFORE UPDATE OF name ON command_sets
FOR EACH ROW
WHEN trim(NEW.name) = '' OR typeof(NEW.name) != 'text'
BEGIN
    SELECT RAISE(ABORT, 'invalid name: name cannot be empty or non-text');
END;

-- Unique index on the trimmed name to enforce uniqueness regardless of surrounding whitespace
CREATE UNIQUE INDEX IF NOT EXISTS idx_command_sets_trimmed_name_unique ON command_sets (trim(name));

-- Additional trigger: Prevent duplicate trimmed names at DB level as a defense-in-depth
CREATE TRIGGER IF NOT EXISTS command_sets_check_name_insert_duplicate
BEFORE INSERT ON command_sets
FOR EACH ROW
WHEN EXISTS(SELECT 1 FROM command_sets WHERE TRIM(name) = TRIM(NEW.name))
BEGIN
    SELECT RAISE(ABORT, 'invalid name: duplicate trimmed name');
END;

CREATE TRIGGER IF NOT EXISTS command_sets_check_name_update_duplicate
BEFORE UPDATE OF name ON command_sets
FOR EACH ROW
WHEN EXISTS(SELECT 1 FROM command_sets WHERE TRIM(name) = TRIM(NEW.name) AND id != OLD.id)
BEGIN
    SELECT RAISE(ABORT, 'invalid name: duplicate trimmed name');
END;
