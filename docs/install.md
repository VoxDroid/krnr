# Installer design — krnr

This document describes a safe, cross-platform installer strategy for `krnr`.
It focuses on a pragmatic, low-risk first implementation (per-user installer CLI) with a path to full platform packaging (MSI/Winget, Homebrew, Scoop, .deb/.rpm).

## Goals
- Offer a friction-free, secure per-user install experience via `krnr install`.
- Avoid requiring admin privileges by default; provide an explicit `--system` mode that requires elevation.
- Be transparent and reversible: provide `--dry-run`, `--uninstall`, and a rollback mechanism.
- Make the implementation testable and CI-friendly.
- Later produce official packages (Homebrew/Scoop/Winget/MSI/.deb/.rpm) using GoReleaser. (in progress: VoxDroid)

## Non-goals (initial)
- Not attempting to replace platform package managers in the first iteration.
- Not implementing auto-updates (defer to package managers for update semantics).

## User stories
- I am a user and want to type `krnr install` and have a runnable `krnr` on my PATH without admin rights.
- I am an advanced user and want `krnr install --system` to install for all users (requires elevation).
- I want `krnr install --dry-run` to show exactly what will change.
- I want `krnr uninstall` to remove what the installer added.

## Quick start — Install & setup (users)
Follow these steps to install and verify `krnr` on your machine. These steps assume you've downloaded or built a `krnr` binary for your platform. For full details and advanced options, see this document.

### Install (per-user)
- Place the binary somewhere you control (recommended: `~/krnr/bin` on Unix/macOS, `%USERPROFILE%\krnr\bin` on Windows).
- Run the installer to copy and optionally add the directory to PATH:
  - `./krnr install --user --from ./krnr --add-to-path` (Unix/macOS)
  - `.\krnr.exe install --user --from .\krnr.exe --add-to-path` (PowerShell on Windows)
- To skip interactive prompts (for scripted installs), provide `--yes` (use with caution):
  - `./krnr install --user --from ./krnr --add-to-path --yes`

#### Installing as `sudo` (user scope)
- If you run `krnr install` under `sudo` but select a **user**-scope install, `krnr` will detect the invoking user via `SUDO_USER` and perform the install for that user (not `root`).
  - The installer writes install metadata into the invoking user's `KRNR_HOME` and ensures the metadata file and installed binary are owned by that user so `krnr status` and `krnr uninstall` work without sudo.
  - `krnr status` will reflect `on PATH: true` if you chose to add the target dir to PATH even before you start a new shell session (status uses recorded metadata to infer the intended state).
  - This behavior keeps sudo->user installs predictable and allows later non-sudo maintenance (status/uninstall) by the original user.

### Verify installation
- Start a new shell (or restart your terminal) so PATH changes take effect.
- Run `krnr status` to confirm `krnr` is installed and whether its directory is on PATH.
- On Unix/macOS you can also run `which krnr` or `command -v krnr`; on Windows use `Get-Command krnr` in PowerShell.

### Manual PATH (if you prefer to avoid the installer)
- Unix/macOS: add the target dir to your shell rc (for example `export PATH="$HOME/krnr/bin:$PATH"`) and reload your shell.
- Windows (PowerShell): add `%USERPROFILE%\krnr\bin` to your user PATH via System Settings, or use `setx`/PowerShell `Set-ItemProperty` (installer handles details and normalization).

### Troubleshooting
- If `krnr` is not found after install, ensure you started a new shell session or your PATH was updated correctly.
- If PATH modifications were not allowed for system-level installs, rerun with elevated privileges (or use a per-user install).
- For uninstall, run `krnr uninstall --yes` to remove installed files non-interactively; use `--dry-run` first to preview actions.

### Official packages & release status
We provide platform packages and release artifacts (created via GoReleaser) alongside the per-user `krnr install` CLI.

Current release packaging status:
- Windows: MSI installer (WiX) is produced and included in GitHub Releases by the release workflow; MSI includes the License Agreement dialog, Start Menu shortcuts, uninstall support, PATH handling, and ARP metadata (contact and links).
- Winget / Scoop / Chocolatey: manifest templates are present in `packaging/windows/`; these contain placeholders for release URLs and SHA values and will be updated on release. CI automation to replace placeholders and open PRs to the respective registries is pending.
- Linux: `.deb` and `.rpm` packages are produced by GoReleaser via `nfpm` in the release pipeline.

How to get an official MSI now:
- Visit the GitHub release for the latest tag and download `krnr-<version>.msi` and run it interactively (double-click or `msiexec /i <msi>`).

Installing via winget / chocolatey / scoop (after manifests are merged):
- Winget: `winget install --id VoxDroid.krnr` (once the PR is merged in winget-pkgs)
- Scoop: `scoop install krnr` (after the bucket PR is merged)
- Chocolatey: `choco install krnr` (after the package is published to Chocolatey.org)

If you'd like, we can automate manifest replacement and PR submission on release; tell us whether to proceed with manifest PR automation in CI.


## Design decisions
- Default behavior is **per-user** install.
  - Target dirs:
    - Linux/macOS: `$HOME/krnr/bin` (per-application directory under the user's home)
    - Windows: `%USERPROFILE%\krnr\bin` (created if needed)
    - macOS can also use `~/Applications` for packaged .app, but initial scope is CLI binary placement.
- PATH handling:
  - Detect whether target dir is already on `PATH` in current shell environment.
  - If not on PATH, show the exact shell rc modification lines and prompt the user to accept; only append to shell rc if user confirms (support `--yes` to skip prompt).
  - On Windows we use PowerShell to persistently set the User PATH by default; `--system` will attempt to set the Machine (system) PATH and **requires elevation**. PowerShell writes use `-EncodedCommand` (UTF-16LE base64) to avoid quoting/escaping issues when setting PATH persistently.
  - The installer records the previous PATH value and will attempt to restore it at uninstall. After any persistent PATH write (both install and uninstall) `krnr` runs a post-write normalization fixer that collapses doubled backslashes and ensures PATH entries remain well-formed — this resolves a class of PATH corruption issues on Windows. Tests and CI use a `KRNR_TEST_NO_SETX` test mode to avoid modifying real environments; set this env var in CI or local tests to prevent persistent PATH changes.

### Uninstall behavior

- `krnr uninstall` reads the recorded install metadata (`~/.krnr/install_metadata.json`) and attempts to undo changes (remove installed binary, restore PATH to recorded previous value when present, or remove the installed directory from PATH otherwise).
- When restoring a recorded full PATH value `krnr` uses the same safe, encoded PowerShell approach and runs the normalization fixer afterwards; if the fixer makes changes its message is included in the uninstall actions so users can see the diagnostics.
- `krnr uninstall` supports `--dry-run` (show planned actions), `--yes` (skip interactive confirmation), and `--verbose` (show before/after PATH details when available). If the binary is in use and cannot be removed, the uninstall will instruct you to run the downloaded `krnr` binary (for example from Downloads) and re-run `krnr uninstall` so the installed executable is not locked by a running process.
- Permissions:
  - `--system` attempts system install paths (`/usr/local/bin`, `C:\Program Files\krnr`) and will require elevation. If not elevated, print guidance and exit with a helpful message.
- Safety & rollback:
  - Before modifying any file (shell rc or system PATH), create backups (e.g., `~/.bashrc.krnr.bak.<timestamp>`).
  - Record an install metadata file in the data dir (e.g., `~/.krnr/install_metadata.json`) that lists changes for uninstall.
  - `krnr uninstall` uses the metadata file to undo changes and restore backups and previous PATH values where possible; it also attempts to remove the install directory if it becomes empty.
  - **Important:** if uninstall fails with "access denied" because the installed binary is in use, run the downloaded `krnr` binary (for example, the copy in your Downloads folder) and run `krnr uninstall` from there so the installed program is not locked by the running process. For system installs you may need to run the downloaded binary elevated (Run as Administrator).
- Binary source:
  - Installer can operate on the current running executable (via `os.Executable()`), or accept a path to a built artifact (`--from <file>`). This supports both dev builds and release artifacts.
- Safety checks:
  - Verify checksum of provided artifact (if `--from` and checksum provided) or verify embedded version before copying.
  - Prompt user before overwriting an existing `krnr` in target location.

## CLI proposal
- `krnr install [--user|--system] [--path <dir>] [--from <file>] [--yes] [--dry-run]`
  - `--user` default installs to user-local bin (default: `~/krnr/bin` or `%USERPROFILE%\krnr\bin`).
  - `--system` install to OS system location (requires elevation).
  - `--path` explicitly set target dir.
  - `--from` path to a binary to install; default is current executable.
  - `--yes` non-interactive accept (dangerous; requires caution in docs).
  - `--dry-run` show exact actions without performing them.
- `krnr uninstall [--path <dir>] [--yes] [--dry-run]` — undo last install performed by the installer; `--dry-run` shows planned actions (recommended before running).
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
6. Create GoReleaser config for packaging and add release CI. (2-3 days) (initial config and release-goreleaser.yml added)

### Packaging validation
- Run `goreleaser` locally in snapshot mode to verify build and packaging output without publishing: `rm -rf dist && goreleaser release --snapshot`.
- Use the `release-validate.yml` workflow (manual `workflow_dispatch` or push a `v*` tag) to produce artifacts in CI and upload them as workflow artifacts for inspection.
- Inspect generated artifacts (`dist/`) for expected package types: `*-portable.*`, Debian `.deb`, RPM `.rpm`, Homebrew artifacts, and Windows installers (ZIP/MSI as produced).
- Verify checksums and signatures (if configured) and confirm package contents and installation paths prior to publishing.
- Note: workflows explicitly remove any existing `dist/` directory before invoking GoReleaser to avoid compatibility issues with older GoReleaser versions that don't support the `--rm-dist` flag.
- After a real release, update Homebrew formula SHA and Scoop/Winget URLs/SHA to point to the released assets.

### Publishing notes
- The integrated `release.yml` runs GoReleaser to **publish artifacts automatically** when a version commit (e.g., `v1.2.3`) is detected.
- Required secrets:
  - `PERSONAL_TOKEN` (recommended) — a GitHub Personal Access Token with `repo`, `releases`, and `packages` write permissions. This token is used by GoReleaser to create the GitHub Release and to push to other repositories (e.g., Homebrew taps or Scoop buckets) if those targets are configured.
  - `GITHUB_TOKEN` (auto-provided) is also available but may be limited for cross-repo operations; use `PERSONAL_TOKEN` if you need to push to other repos.
- If you want automatic publishing to Homebrew, Scoop, or Winget, ensure the token has write access to those target repos (or provide separate tokens via secrets).
- We recommend running `release-validate.yml` first to inspect generated artifacts before creating an actual release.

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
