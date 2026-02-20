# v1.2.8 - 2026-02-20

This patch improves installer robustness when the installer is executed under `sudo` but a per-user install is requested. It ensures recorded metadata and installed files are visible and removable by the original invoking user.

## Bug fixes

### Correct metadata ownership for sudo→user installs
When `krnr install` runs under `sudo` and the user chooses a **user** install, metadata and PATH modifications were sometimes recorded under root and not visible to the invoking user.

**Fix:** the installer writes a copy of the install metadata into the original user's `~/.krnr` (when `SUDO_USER` is set) and ensures the metadata file and related directories are owned by that user. This allows `krnr status` and `krnr uninstall` to work when run later as the normal user.

### Ensure installed binary ownership and uninstallability
Binaries installed by root into a user's directory are now chowned to the target user so subsequent non-root `krnr uninstall` works as expected.

### Status & reinstall fixes
`krnr status` now respects recorded `AddedToPath` so it reports `on PATH: true` immediately after add-to-PATH installs. Reinstall-after-uninstall under `sudo` no longer leaves stale or unreadable metadata.

## Tests & docs
- Added unit tests for SUDO_USER install/uninstall/reinstall and metadata-based status inference.
- Documented sudo→user installer behavior in `docs/install.md`.

## Upgrade notes
No DB or CLI-breaking changes. Upgrade to v1.2.8 to receive installer fixes.
