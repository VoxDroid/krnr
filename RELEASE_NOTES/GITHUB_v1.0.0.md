# krnr v1.0.0 — First official release (2026-01-11)

krnr v1.0.0 is the project's first official stable release: it focuses on stability, packaging, and a set of productivity features that improve discoverability and organization of saved command sets.

## Highlights

- Tagging & search
  - `krnr tag add|remove|list` to manage tags on command sets
  - `krnr list --tag <tag>` to filter results by tag
- Text & fuzzy search
  - `krnr list --filter <text>` performs substring search across name, description, commands, and tags
  - `krnr list --filter <text> --fuzzy` enables case-insensitive subsequence fuzzy matching (helpful for short queries and typos)
- Parameters & variable substitution
  - Use `{{param}}` in saved commands and provide values at runtime via `--param name=value`
  - Support for `env:VAR` to load values from environment variables; missing parameters are prompted interactively
- Versioning & history
  - Snapshots of command sets via `command_set_versions`
  - `krnr history <name>` and `krnr rollback <name> --version <n>` to inspect and restore previous states
- Security & safety
  - Conservative destructive command checks (use `--force` to override) and redaction of secret-like params
  - Confirmations by default for `krnr delete` and `krnr install` (use `--yes` to automate)
- Packaging & installers
  - Windows MSI (WiX) and packaging manifests for Homebrew, Scoop, Winget, and Chocolatey prepared for release automation
- Tests & docs
  - Unit and integration tests added for the above features; `docs/cli.md` updated with examples

## Upgrade notes

- No DB schema migrations are required; v1.0.0 is backward compatible with previous registries.
- Recommended: run the test suite and review `docs/cli.md` examples after upgrading.

## Assets

Release artifacts include platform installers and archives (MSI on Windows, deb/rpm via nfpm, Homebrew/Scoop/Winget manifests). See the Releases page for published assets.

---

For full details, changelog, and contributors, see `docs/releases/v1.0.0.md` and `CHANGELOG.md`.
