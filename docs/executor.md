# Execution Engine

The execution engine provides an `Executor` type used to run shell commands in an
OS-aware manner.

Key points:

- Uses `cmd /C <cmd>` on Windows by default and `bash -c <cmd>` on Unix-like
  systems.
- Supports `DryRun` (no execution) and `Verbose` (prints dry-run messages).
- Execution streams stdout and stderr to provided writers so tests and callers
  can capture or forward output.

API:
- `Executor{DryRun: bool, Verbose: bool, Shell: string}`
- `Execute(ctx, command, cwd, stdout, stderr) error`

Notes:
- By default, `Shell` is empty and the OS default shell is used. Set `Shell` to
  `pwsh` to use PowerShell Core if you prefer.
