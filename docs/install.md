# Installer design — krnr

This document describes a safe, cross-platform installer strategy for `krnr`.
It focuses on a pragmatic, low-risk first implementation (per-user installer CLI) with a path to full platform packaging (MSI/Winget, Homebrew, Scoop, .deb/.rpm).

## Goals
- Offer a friction-free, secure per-user install experience via `krnr install`.
- Avoid requiring admin privileges by default; provide an explicit `--system` mode that requires elevation.
- Be transparent and reversible: provide `--dry-run`, `--uninstall`, and a rollback mechanism.
- Make the implementation testable and CI-friendly.
- Later produce official packages (Homebrew/Scoop/Winget/MSI/.deb/.rpm) using GoReleaser.

## Non-goals (initial)
- Not attempting to replace platform package managers in the first iteration.
- Not implementing auto-updates (defer to package managers for update semantics).

## User stories
- I am a user and want to type `krnr install` and have a runnable `krnr` on my PATH without admin rights.
- I am an advanced user and want `krnr install --system` to install for all users (requires elevation).
- I want `krnr install --dry-run` to show exactly what will change.
- I want `krnr uninstall` to remove what the installer added.

## Design decisions
- Default behavior is **per-user** install.
  - Target dirs:
    - Linux/macOS: `$HOME/.local/bin` (or `$HOME/bin` on macOS if `~/.local/bin` not preferred)
    - macOS can also use `~/Applications` for packaged .app, but initial scope is CLI binary placement.
    - Windows: `%USERPROFILE%\bin` (created if needed) or user's AppData bin dir.
- PATH handling:
  - Detect whether target dir is already on `PATH` in current shell environment.
  - If not on PATH, show the exact shell rc modification lines and prompt the user to accept; only append to shell rc if user confirms (support `--yes` to skip prompt).
  - For Windows, use `setx` for persistent user PATH updates; warn about requiring re-login to apply.
- Permissions:
  - `--system` attempts system install paths (`/usr/local/bin`, `C:\Program Files\krnr`) and will require elevation. If not elevated, print guidance and exit with a helpful message.
- Safety & rollback:
  - Before modifying any file (shell rc or system PATH), create backups (e.g., `~/.bashrc.krnr.bak.<timestamp>`).
  - Record an install metadata file in the data dir (e.g., `~/.krnr/install.json`) that lists changes for uninstall.
  - `krnr uninstall` uses the metadata file to undo changes and restore backups where possible.
- Binary source:
  - Installer can operate on the current running executable (via `os.Executable()`), or accept a path to a built artifact (`--from <file>`). This supports both dev builds and release artifacts.
- Safety checks:
  - Verify checksum of provided artifact (if `--from` and checksum provided) or verify embedded version before copying.
  - Prompt user before overwriting an existing `krnr` in target location.

## CLI proposal
- `krnr install [--user|--system] [--path <dir>] [--from <file>] [--yes] [--dry-run]`
  - `--user` default installs to user-local bin.
  - `--system` install to OS system location (requires elevation).
  - `--path` explicitly set target dir.
  - `--from` path to a binary to install; default is current executable.
  - `--yes` non-interactive accept (dangerous; requires caution in docs).
  - `--dry-run` show exact actions without performing them.
- `krnr uninstall [--path <dir>] [--yes]` — undo last install performed by the installer.
- `krnr install --check` — diagnostics: show installability (path permissions, PATH presence).

## Tests
- Unit tests for path resolution, rc file patch generation, metadata creation.
- Integration tests (in CI, where possible) run `krnr install --dry-run --from <tmp>` and assert the actions reported are correct.
- E2E smoke: in a container or ephemeral VM, perform `krnr install --user --from <artifact>` then `krnr run <sample>` to verify binary works.
- Make tests safe for CI (do not actually alter real user files; use a fake HOME/PATH environment and verify changes affect those simulated files).

## Implementation plan (phases)
1. Design and docs (this doc) — define behavior and acceptance criteria. (Done)
2. CLI scaffold: `cmd/install.go` with `--dry-run` and `--user` behaviors; unit tests. (2-3 dev hours)
3. Implement file operations (copy, set executable, create backups, metadata file). Add uninstall. (1-2 days)
4. Add integration tests & CI job that runs dry-run tests on all OS runners. (1 day)
5. Add `--system` with conditional elevation guidance and tests. (1 day)
6. Create GoReleaser config for packaging and add release CI. (2-3 days)

## Acceptance criteria
- `krnr install --dry-run` shows changes without modifying user files.
- `krnr install --user` places executable in the per-user directory and prints exact PATH change instructions (makes the change only if user consents or `--yes` provided).
- `krnr uninstall` reverses changes created by a prior install.
- Unit & integration tests cover core behaviors; CI runs a dry-run across platforms.

## Security considerations
- Never auto-elevate without explicit user action.
- Always require user confirmation before system-level changes.
- Document the exact lines added to shell files; allow users to opt-out.
- If installing from a remote artifact (future), require checksum/signature verification.

## Open questions
- macOS signed installer (not in initial scope) — do we eventually want signed installers or notarization? (Recommended for public release.)
- Which exact per-user location on macOS do we prefer? (`~/bin` vs `~/.local/bin`) — choose consistent cross-platform default (use `$HOME/.local/bin` for consistency).

---

If this looks good, I'll implement the CLI scaffold and add unit tests for `--dry-run` and path-shell editing behaviors next. Otherwise tell me what to adjust in the design.