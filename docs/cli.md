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

Edit a command set. Use `-c` multiple times to replace commands non-interactively; if no `-c` is provided the user's editor (from `$EDITOR`) will be opened to edit the command list.

## delete

`krnr delete <name> [--confirm]`

Delete a command set; use `--confirm` to prompt interactively before deleting.
