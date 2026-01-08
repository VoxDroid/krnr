# CLI Commands

This document describes the top-level CLI commands and usage.

## save

`krnr save <name> -c "echo hello" -c "echo world" -d "description" --author "Name" --author-email "a@example.com"`

Saves a named command set with provided commands (use `-c` multiple times).

Author metadata:

- `--author` (`-a`) sets the author name for the saved command set and overrides any stored identity.
- `--author-email` (`-e`) optionally sets the author email.
- If `--author` is not provided, `krnr save` will use the stored `whoami` identity if present.

## whoami

`krnr whoami set --name "Your Name" [--email "you@example.com"]` — store a default author identity for future `save` operations.

`krnr whoami show` — display the stored identity.

`krnr whoami clear` — remove the stored identity.


## list

`krnr list`

Lists saved command sets.

## describe

`krnr describe <name>`

Shows details of a command set and its commands.

## run

`krnr run <name> [--dry-run] [--confirm] [--verbose] [--shell <shell>]`

Runs the commands in order. Defaults to stopping on the first failing command.
Use `--dry-run` to preview commands without running them. `--confirm` will
prompt interactively before running.

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

Examples:

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

Record commands from standard input into a new command set. After running the command, type commands one per line and finish with EOF (Ctrl-D on Unix, Ctrl-Z on Windows). Blank lines and lines beginning with `#` are ignored. The recorded commands will be saved as a new command set named `<name>`.
## delete

`krnr delete <name> [--confirm]`

Delete a command set; use `--confirm` to prompt interactively before deleting.
