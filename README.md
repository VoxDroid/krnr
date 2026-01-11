# krnr — Kernel Runner

[![CI](https://github.com/VoxDroid/krnr/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/VoxDroid/krnr/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/VoxDroid/krnr?label=release)](https://github.com/VoxDroid/krnr/releases) [![Downloads](https://img.shields.io/github/downloads/VoxDroid/krnr/total?label=downloads&color=blue)](https://github.com/VoxDroid/krnr/releases) [![Go Version](https://img.shields.io/github/go-mod/go-version/VoxDroid/krnr?label=go)](https://github.com/VoxDroid/krnr) [![License](https://img.shields.io/github/license/VoxDroid/krnr)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/VoxDroid/krnr)](https://goreportcard.com/report/github.com/VoxDroid/krnr) [![pkg.go.dev](https://pkg.go.dev/badge/github.com/VoxDroid/krnr)](https://pkg.go.dev/github.com/VoxDroid/krnr)

krnr is a cross-platform CLI that provides a global, persistent registry of named terminal workflows (command sets) backed by SQLite. It helps you save, re-run, and share commonly used shell sequences across machines.

---

## Table of contents

- [Quick summary](#quick-summary)
- [Badges & Status](#badges--status)
- [Install & Setup](#install--setup)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Configuration & Database](#configuration--database)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License & Credits](#license--credits)

---

## Quick summary

- Use `krnr save <name>` to persist a named command set (multiple commands per name).
- Use `krnr run <name>` to execute a saved set safely and consistently.
- Store author metadata with `krnr whoami` for recorded runs.

For a longer introduction and design notes, see `krnr_docs/PROJECT_OVERVIEW.md` and `docs/architecture.md`.


## What's new — v1.0.0 (First official release) 🔖

v1.0.0 is the project's first official stable release (2026-01-11). Key improvements include:

- Tagging & discovery: `krnr tag` and `krnr list --tag` for organizing and filtering command sets.
- Better search: `krnr list --filter <text>` and `--fuzzy` for substring and fuzzy subsequence matches.
- Parameters & env sourcing: `{{param}}` substitution, `--param name=value`, and `env:VAR` support.
- Versioning & history: `krnr history <name>` and `krnr rollback <name> --version` to manage snapshots.
- Packaging & installers: Windows MSI (WiX) and package manifests for Homebrew/Scoop/Winget; release automation produces installers and archives.
- Security & safety: conservative destructive checks, parameter redaction, and confirmation prompts.

See the full release notes: `docs/releases/v1.0.0.md` and the detailed changelog in `CHANGELOG.md`.

---

## Badges & Status

- CI: GitHub Actions (format, lint, test, build)
- Releases & downloads: GitHub Releases
- Code quality: Go Report Card, `pkg.go.dev`
- License: MIT (see `LICENSE`)

---

## Install & Setup

### Prebuilt releases (recommended)

Download for your platform from the Releases page: https://github.com/VoxDroid/krnr/releases

### Homebrew (macOS / Linux)

If you have Homebrew installed, the repository includes a formula under `packaging/homebrew/krnr.rb`. Example:

```bash
brew install --formula=packaging/homebrew/krnr.rb
```

### Windows package managers

- Winget / Scoop manifests are included in `packaging/windows/winget` and `packaging/scoop` respectively.

### Manual install (local binary)

Place an executable on your PATH or use the bundled installer:

Unix / macOS

```bash
./krnr install --user --from ./krnr --add-to-path
```

PowerShell / Windows

```powershell
.
\krnr.exe install --user --from .\krnr.exe --add-to-path
```

See `docs/install.md` for full installation guidance and PATH handling details.

---

## Quick Start

Developer quick start

```bash
# Build locally
Go 1.25.5+ is recommended
go build -v -o krnr .
# Run interactively
./krnr --help
```

User quick start

```bash
# Save a simple command set
krnr save hello -d "Greet" -c "echo Hello"
# Run it
krnr run hello
```

Interactive recording

Use `krnr record <name>` to type commands directly; finish recording by entering `:end` (aliases `:save` and `:quit`).

---

## Commands (summary table)

| Command | Description | Example |
|---|---|---|
| `krnr install` | Install the `krnr` binary to your system or user PATH | `krnr install --user --from ./krnr --add-to-path` |
| `krnr uninstall` | Uninstall krnr (remove binary and PATH modifications) | `krnr uninstall --user` |
| `krnr status` | Show installation status (user and system) | `krnr status` |
| `krnr save <name>` | Save a named command set | `krnr save build -d "Build project" -c "go build ./..."` |
| `krnr record <name>` | Record commands interactively into a new command set | `krnr record demo` (then type commands, `:end` to finish) |
| `krnr run <name>` | Run a named command set | `krnr run hello --confirm` |
| `krnr list` | List saved command sets | `krnr list` |
| `krnr describe <name>` | Show details for a command set | `krnr describe hello` |
| `krnr edit <name>` | Edit a command set using your editor | `krnr edit hello` |
| `krnr delete <name>` | Delete a command set | `krnr delete obsolete` |
| `krnr whoami` | Manage stored author identity (`set`, `show`, `clear`) | `krnr whoami set --name "Alice" --email alice@example.com` |
| `krnr version` | Print version information | `krnr version` |

For more flags and options, run `krnr <command> --help` or see the `cmd/` directory.

---

## Configuration & Database

- Default DB location: `$HOME/.krnr/krnr.db`
- Environment variables:
  - `KRNR_HOME` — override the data directory
  - `KRNR_DB` — set full DB file path (overrides KRNR_HOME)

Migrations and schema are bundled and applied automatically on first run. See `docs/config.md` and `docs/database.md` for advanced configuration and backup/restore tips.

---

## Development & Testing

- Format: `./scripts/fmt.sh`
- Lint: `./scripts/lint.sh` (use Docker fallback on Windows if needed)
- Unit tests: `go test ./...`
- DB tests: `go test ./internal/db -v`

Building a local release:

```bash
./scripts/release.sh v0.1.0
```

---

## Troubleshooting & Notes

- If `golangci-lint` fails on Windows with export-data errors, use the Dockerized linter (see above).
- If install removes or doesn't add to PATH as expected, check `krnr status` and refer to `docs/install.md`.

---

## Contributing

Contributions are welcome — please see `CONTRIBUTING.md` for the full guidelines (create an issue first if in doubt). Quick pointers:

- Development: Go 1.25.5+ is recommended; build with `go build`.
- Recommended tooling: `golangci-lint` v2.8.0 (use `scripts/lint.sh` to run it locally or the Docker fallback).
- Formatting & linting: run `./scripts/fmt.sh` and `./scripts/lint.sh` before opening a PR.
- Tests: run `go test ./...` and add tests for new behavior.
- Use the `.github` issue templates and the PR template when opening issues or pull requests.

## Community & Support

- Documentation and developer guides live in `docs/` and `README.md`.
- For help, ask a question using the **Support Question** issue template or the Discussions tab: https://github.com/VoxDroid/krnr/discussions

## Security

If you discover a security vulnerability, follow `SECURITY.md`: we prefer private reports by email to [izeno.contact@gmail.com](mailto:izeno.contact@gmail.com) to avoid public disclosure until a fix is available.

## Code of Conduct

Please follow `CODE_OF_CONDUCT.md`. If you believe someone has violated the code, contact the maintainers at [izeno.contact@gmail.com](mailto:izeno.contact@gmail.com) or open a private issue labeled "Code of Conduct Violation."

---

## License & Credits

krnr is open-source and licensed under the MIT License — see `LICENSE`.

---

If you'd like, I can open a draft PR with these changes and also add the commands table as an auto-generated snippet under `docs/` or keep it only in the README. Which do you prefer?
