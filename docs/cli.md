# CLI Commands

This document describes the top-level CLI commands and usage.

## save

`krnr save <name> -c "echo hello" -c "echo world" -d "description"`

Saves a named command set with provided commands (use `-c` multiple times).

## list

`krnr list`

Lists saved command sets.

## describe

`krnr describe <name>`

Shows details of a command set and its commands.

## run

`krnr run <name> [--dry-run] [--confirm] [--verbose]`

Runs the commands in order. Defaults to stopping on the first failing command.
Use `--dry-run` to preview commands without running them. `--confirm` will
prompt interactively before running.

## edit

`krnr edit <name> [-c "cmd" ...]`

Edit a command set. Use `-c` multiple times to replace commands non-interactively; if no `-c` is provided the user's editor (from `$EDITOR`) will be opened to edit the command list interactively.

Interactive edit details:

- The editor will be pre-populated with the command set, one command per line.
- Blank lines and lines beginning with `#` are ignored when saving (use `#` for comments).
- The `EDITOR` environment variable is respected; if unset, a sensible platform default is used (`notepad` on Windows, `vi` on Unix).

## delete

`krnr delete <name> [--confirm]`

Delete a command set; use `--confirm` to prompt interactively before deleting.
