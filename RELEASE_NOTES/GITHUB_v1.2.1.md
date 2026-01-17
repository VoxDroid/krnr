# v1.2.1 — 2026-01-17

This patch release fixes Windows quoting and CLI save robustness issues:

- `krnr save`: improved handling when shells split quoted arguments — joins or merges leftover tokens into a single command and heuristically re-quotes `findstr /C:"..."` patterns when appropriate.
- Executor (Windows): directly handles simple `<left> | findstr ...` pipelines by executing the left command and piping its stdout into `findstr` (avoids `cmd.exe` quoting pitfalls).
- Includes unit tests for both save and findstr pipeline scenarios, adds robust shell-token splitting (via `github.com/kballard/go-shellquote`), and cleans up linter/complexity issues.

Upgrade guidance: Simply upgrade to v1.2.1; no DB schema changes are required. If you rely heavily on complex Windows shell invocations, prefer quoting `-c` values fully or using a command file for maximum stability.
