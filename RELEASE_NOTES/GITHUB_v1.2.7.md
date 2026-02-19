# v1.2.7 - 2026-02-19

This patch prevents passwords from being visible when running interactive commands (for example `sudo`) via `krnr` or the TUI by disabling local echo on the host terminal while a PTY-backed child is running.

## Bug fixes

### Hide password input during interactive runs
Interactive programs (such as `sudo`) already read from the child's PTY, but the host terminal could locally echo typed keystrokes — which made passwords visible to observers of the host terminal.

**Fix:** temporarily disable the host terminal's ECHO flag while the PTY-backed child runs and restore the previous terminal state afterwards. The change only toggles local echo (preserves output post-processing) to avoid affecting child output rendering.

## Tests & docs
- Unit tests added to validate host-terminal echo toggling in PTY scenarios.
- Documentation updated to mention the behavior in `docs/executor.md` and `docs/tui.md`.

## Upgrade notes
No DB or user-facing CLI changes required — upgrade to v1.2.7 to get the fix.
