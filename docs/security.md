# Security & Safety

krnr defaults to **safe execution**: it will not run obviously destructive commands unless explicitly overridden.

Key behaviors

- `krnr run` performs a conservative check against a list of dangerous command patterns (e.g., `rm -rf /`, `dd if=`, `mkfs`, fork bombs). If a command looks unsafe the run will be refused with a message.
- Use `--force` to override the safety check when you know the command is intentional.
- Use `--dry-run` and `--confirm` to preview and confirm runs.

Guidance

- The safety check is conservative: it aims to stop obviously destructive commands but may not catch every harmful command. Always review commands before running.
- For automation you can use `--force` in trusted environments but prefer not to embed dangerous operations in shared command sets.
