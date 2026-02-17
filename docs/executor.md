# Execution Engine

The execution engine provides an `Executor` type used to run shell commands in an
OS-aware manner.

Key points:

- Uses `cmd /C <cmd>` on Windows by default and `bash -c <cmd>` on Unix-like
  systems.
- Supports `DryRun` (no execution) and `Verbose` (prints dry-run messages).
- Execution streams stdout and stderr to provided writers **live** via
  `io.MultiWriter` so output appears immediately (no buffering until
  completion).

API:
- `Executor{DryRun: bool, Verbose: bool, Shell: string}`
- `Execute(ctx, command, cwd, stdin, stdout, stderr) error` (accepts an explicit stdin reader so callers can provide input for interactive commands or PTY-backed sessions)

Hybrid PTY mode:
- When the provided `stdin` exposes a terminal file descriptor (via `Fd()`) on
  Unix-like platforms, the executor uses a **hybrid PTY**: the child's stdin and
  controlling terminal use a PTY (so programs like `sudo` that open `/dev/tty`
  work), while stdout and stderr remain as pipes. This lets interactive prompts
  be answered while keeping output simple and viewport-friendly.
- PTY output written to `/dev/tty` by the child (e.g., password prompts) is read
  from the PTY master and forwarded to the caller's stdout.

Notes:
- By default, `Shell` is empty and the OS default shell is used. Set `Shell` to
  `pwsh` to use PowerShell Core if you prefer.
