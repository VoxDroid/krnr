# v1.2.2 — 2026-01-17

Patch release focusing on a robustness fix for the `save` command.

What’s changed

- Preserve comma-containing commands passed via a single `-c` flag (PowerShell `Select-Object` usage). A single quoted `-c` will now be stored as a single command and is no longer split on commas.
- Internal: switched the repeatable command flag to `StringArray` to avoid unintended splitting and added tests to cover the behavior.

Upgrade notes

- No DB schema changes are required.
- When using Windows shells, prefer quoting `-c` values fully (e.g., `-c 'Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture'`) to avoid shell-level tokenization issues.

Thanks to contributors and maintainers for quick turnaround on this fix.
