# CLI Commands

This document describes the top-level CLI commands and usage.

## save

`krnr save <name> -c "echo hello" -c "echo world" -d "description" --author "Name" --author-email "a@example.com"`

Saves a named command set with provided commands (use `-c` multiple times).

If the provided name already exists, `krnr save` will warn and prompt you to enter a different name (interactive) instead of failing with a DB constraint error.

Author metadata:

- `--author` (`-a`) sets the author name for the saved command set and overrides any stored identity.
- `--author-email` (`-e`) optionally sets the author email.
- If `--author` is not provided, `krnr save` will use the stored `whoami` identity if present.

## whoami

`krnr whoami set --name "Your Name" [--email "you@example.com"]` — store a default author identity for future `save` operations.

`krnr whoami show` — display the stored identity.

`krnr whoami clear` — remove the stored identity.


## list

`krnr list [--tag <tag>] [--filter <text>] [--fuzzy]`

Lists saved command sets.

Flags:

- `--tag <tag>` — filter results to command sets that have the given tag
- `--filter <text>` — text search against name, description, commands, and tags (substring match by default)
- `--fuzzy` — enable fuzzy matching for `--filter` (case-insensitive subsequence matching)

Examples:

- `krnr list --tag utils`
- `krnr list --filter demo`
- `krnr list --filter dmo --fuzzy`  # fuzzy-matches `demo`

## tui

`krnr tui`

Starts the interactive terminal UI (TUI) prototype. The TUI provides a keyboard-first interface for browsing command sets.

Shortcuts:

- `?` — show help
- `q` or `Esc` — quit
- `Enter` — show details for the selected command set

This prototype is implemented using Bubble Tea (`github.com/charmbracelet/bubbletea`) and is intended as an iterative starting point for further UX and feature work.

Note: The TUI is a long-term initiative (v1.2.0). The goal is to make `krnr tui` a fully-supported, accessible interface that allows interactive browsing and runs while keeping the CLI as the canonical, scriptable automation surface. See `CHANGELOG.md` (v1.2.0 planned) and `docs/releases/v1.2.0.md` for the initiative plan.

## describe

`krnr describe <name>`

Shows details of a command set and its commands.

## history

`krnr history <name>`

Shows version history for a named command set. Each row includes the version number, timestamp, operation (create/update/delete/rollback), and author when present.

Examples:

- `krnr history hello`

## rollback

`krnr rollback <name> --version <n>`

Rollback a command set to the specified version. Rollback replaces the current commands with the snapshot from the requested version and records the rollback as a new version.

Examples:

- `krnr rollback hello --version 2`

## export

`krnr export db --dst <file>`

Export the entire active database to a portable SQLite file at `<file>`; this performs a WAL checkpoint to ensure a consistent copy. Use `krnr export set <name> --dst <file>` to export a single command set (an "entry") into a minimal SQLite file that contains only that set and its commands.

Running `krnr export` with no arguments launches an interactive prompt where you can choose `db` or `set`, provide a destination path (or accept a sane default for DB exports), and confirm actions.

Examples:

- `krnr export db --dst ~/krnr-backup.db`
- `krnr export set my-entry --dst ./my-entry.db`
- `krnr export` (interactive mode)
## import

`krnr import db <file> [--overwrite] [--on-conflict=rename|skip|overwrite|merge] [--dedupe]`

Import the provided `<file>` as the active database. Use `--overwrite` to replace the active DB file. Alternatively, specify a per-set conflict policy with `--on-conflict` to merge or handle conflicts when importing into an existing DB (for example `--on-conflict=merge --dedupe`).

`krnr import set <file> [--on-conflict=rename|skip|overwrite|merge] [--dedupe]`

Import a minimal exported command set file (created by `krnr export set`) into the active DB. Use `--on-conflict` to control how name collisions are handled:

- `rename` (default) — append `-import-N` to avoid overwrites
- `skip` — do not import a set when a name collision exists
- `overwrite` — delete the existing set and insert the imported set
- `merge` — append incoming commands into the existing set; use `--dedupe` to remove exact-duplicate commands when merging

Running `krnr import` with no arguments launches an interactive prompt to choose `db` or `set` and walk through options.

Examples:

- `krnr import db ~/krnr-backup.db --overwrite`
- `krnr import db ~/krnr-backup.db --on-conflict=merge --dedupe`
- `krnr import set ./my-entry.db --on-conflict=merge --dedupe`
- `krnr import` (interactive mode)
## run

`krnr run <name> [--dry-run] [--confirm] [--verbose] [--shell <shell>] [--param <name>=<value>]`

Runs the commands in order. Defaults to stopping on the first failing command.
Use `--dry-run` to preview commands without running them. `--confirm` will
prompt interactively before running.

The `--param` (short `-p`) flag allows passing named parameters into the
commands. Use `--param` multiple times for multiple parameters (for example
`-p user=alice -p token=env:API_TOKEN`). Parameter values support an
`env:VAR` form to read values from environment variables, and if no value is
provided the CLI will prompt interactively for the parameter value.

Use `--shell` to select the shell used to execute commands (for example
`pwsh`, `powershell`, `bash`, or `cmd`). If omitted, platform defaults are used
(`cmd` on Windows, `bash` on Unix-like systems).

Behavior and notes:

- `--shell pwsh` runs PowerShell Core with `pwsh -Command <cmd>` (requires
  `pwsh` to be installed and on PATH).
- `--shell powershell` on **Windows** will prefer the legacy Windows
  PowerShell executable (`powershell`) if found; otherwise it falls back to
  `pwsh` if available. On non-Windows systems `powershell` will choose
  `pwsh` (the cross-platform implementation).
- Other values are passed through as the executable name and invoked with
  `-c` (e.g., `--shell bash` → `bash -c "..."`).
- If the requested shell executable is not present on `PATH`, execution will
  fail with an "executable file not found" error from the OS. Use
  `where pwsh` (or `Get-Command pwsh`) to check availability on Windows.
- `krnr run` performs a conservative safety check and will refuse to run
  obviously destructive commands (e.g., `rm -rf /`) unless `--force` is used; use `--dry-run` and `--confirm` to preview actions safely.

Examples:

- `krnr run hello --param user=alice --param token=env:API_TOKEN`
- `krnr run hello --dry-run --param release=1.2.3`
- `krnr run hello --shell pwsh` — run with PowerShell Core
- `krnr run hello --shell powershell` — prefer Windows PowerShell on Windows
- `krnr run hello --shell cmd` — force Windows `cmd.exe`
- Omit `--shell` to use sensible platform defaults.

## edit

`krnr edit <name> [-c "cmd" ...]`

Edit a command set. Use `-c` multiple times to replace commands non-interactively; if no `-c` is provided the user's editor (from `$EDITOR`) will be opened to edit the command list interactively.

Developer note — Clean rebuild

- If you make code changes and want to ensure a fresh binary is used when testing CLI behavior, perform a clean rebuild (see `README.md` Clean rebuild (dev) section). On Windows, explicitly build to `krnr.exe` and run `.\krnr.exe run <name>` to verify runtime fixes (for example, output normalization on Windows).
Interactive edit details:

- The editor will be pre-populated with the command set, one command per line.
- Blank lines and lines beginning with `#` are ignored when saving (use `#` for comments).
- The `EDITOR` environment variable is respected; if unset, a sensible platform default is used (`notepad` on Windows, `vi` on Unix).

## record

`krnr record <name> [-d "description"]`

Record commands from standard input into a new command set. After running the command, type commands one per line and finish with EOF (Ctrl-D on Unix, Ctrl-Z on Windows) or use a sentinel to stop recording. Blank lines and lines beginning with `#` are ignored. The recorded commands will be saved as a new command set named `<name>`. If the provided name already exists, `krnr` will warn and prompt you to enter a different name before recording.

Sentinel to end recording: type `:end` on a line by itself to stop recording immediately. Aliases `:save` and `:quit` are also accepted. Sentinel lines are not saved as commands.

## delete

`krnr delete <name> [--yes]`

Delete a command set; an interactive y/n confirmation will be requested by default. Use `--yes` to skip prompts when running non-interactively (for example, in scripts).

## install

`krnr install [--user|--system] [--path <dir>] [--from <file>] [--add-to-path] [--yes] [--dry-run]`

Install the `krnr` binary. By default this performs a per-user install (creates `~/krnr/bin` or `%USERPROFILE%\krnr\bin` on Windows). Use `--system` to install to a system-wide directory (requires elevation). `--path` overrides the target installation directory; `--from` points to a binary to install (defaults to the running executable).

- `--add-to-path` will persistently add the installation directory to PATH (on Windows this will modify the User or Machine PATH depending on `--system`).
- `--dry-run` shows planned actions without changing persistent state; `--yes` accepts prompts non-interactively.
- On Windows, persistent PATH writes use PowerShell `-EncodedCommand` (UTF-16LE base64) to avoid quoting problems; `krnr` runs a post-write normalization fixer that corrects doubled backslashes.
- Tests/CI: set `KRNR_TEST_NO_SETX=1` to avoid persisting PATH changes during tests.

## uninstall

`krnr uninstall [--path <dir>] [--yes] [--dry-run] [--verbose]`

Uninstall the previously installed `krnr` (reads metadata to determine what to remove and how to restore PATH). `--dry-run` shows planned actions; `--verbose` prints diagnostic information including before/after PATH values when available. When a full PATH restore is performed the same safe encoded-PowerShell approach is used and the post-write normalization fixer runs; any fixer message will be included in the uninstall actions to make fixes visible.

## status

`krnr status`

Reports whether `krnr` is installed for the current user and system, and whether the user installation directory is present on the process PATH. Useful for debugging and CI checks.
