# krnr — Kernel Runner

[![CI](https://github.com/VoxDroid/krnr/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/VoxDroid/krnr/actions/workflows/ci.yml) [![Release](https://img.shields.io/github/v/release/VoxDroid/krnr?label=release)](https://github.com/VoxDroid/krnr/releases) [![Downloads](https://img.shields.io/github/downloads/VoxDroid/krnr/total?label=downloads&color=blue)](https://github.com/VoxDroid/krnr/releases) [![Go Version](https://img.shields.io/github/go-mod/go-version/VoxDroid/krnr?label=go)](https://github.com/VoxDroid/krnr) [![License](https://img.shields.io/github/license/VoxDroid/krnr)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/VoxDroid/krnr)](https://goreportcard.com/report/github.com/VoxDroid/krnr) [![pkg.go.dev](https://pkg.go.dev/badge/github.com/VoxDroid/krnr)](https://pkg.go.dev/github.com/VoxDroid/krnr)

**krnr** is a high-performance, cross-platform CLI that provides a **global, persistent registry of named terminal workflows** backed by SQLite. It eliminates the need for managing scattered shell scripts and non-portable aliases by centralizing your command sets in a versioned, durable database accessible from any directory.

---

## Why krnr?

Modern terminal users handle complex sequences across different project roots and operating systems. Standard solutions fall short:

*   **Shell Scripts**: Require file management, PATH modification, and lack structured metadata.
*   **Aliases**: Non-portable across machines and invisible within automated workflows.
*   **Task Runners**: Usually scoped to a specific project directory (e.g., `Makefile`, `package.json`).

**krnr** introduces a **global execution layer**:

*   **Global Access**: Call your workflows from any path without searching for script files.
*   **Industrial Infrastructure**: Powered by SQLite for ACID compliance, versioning, and instant full-text search.
*   **Safety First**: Built-in dry-runs, interactive confirmations, and execution safety checks.
*   **Portable Logic**: Export and import workflows via single-file SQLite "entries" to share across your team.

---

## Key Features

- **Interactive Recording**: Capture complex terminal sessions in real-time with `krnr record`.
- **Dynamic Parameterization**: Inject variables at runtime using `--param key=value` or interactive prompts.
- **Auto-Versioning**: Every modification creates a version snapshot, allowing for instant rollbacks.
- **Native Shell Integration**: Intelligently selects the best shell (Bash, PowerShell Core, CMD) for your OS while allowing manual overrides.
- **Fuzzy Search & Tagging**: Organize hundreds of workflows with metadata-aware search and categorization.

---

## Quick Start

### 1. Installation
Install the binary and add it to your PATH automatically:

```bash
# Unix / macOS
./krnr install --user --add-to-path

# PowerShell / Windows
.\krnr.exe install --user --add-to-path
```

### 2. Save a Workflow
Persist a multi-step sequence with a single name:

```bash
krnr save deploy -d "Production deploy" -c "go test ./..." -c "go build -o app" -c "docker push registry/app"
```

### 3. Execute Anywhere
Run your saved workflow from any directory:

```bash
krnr run deploy --confirm
```

---

## Technical Architecture

krnr acts as a management layer between the user and the operating system's native shells.

### System Flow
![System Flow](assets/svg/system-flow.svg)

### Execution Sequence
![Execution Sequence](assets/svg/execution-sequence.svg)

### Data Model
![Data Model](assets/svg/data-model.svg)

---

## Advanced Usage

### Runtime Parameters
You can use parameters in your commands to make workflows dynamic.

1. **Save a parameterized set**:
   `krnr save config -c "echo Current target: {{target}}"`

2. **Run with explicit value**:
   `krnr run config --param target=production`

3. **Run with Environment Variable**:
   `krnr run config --param target=env:DEPLOY_TARGET`

4. **Interactive Prompt**:
   `krnr run config` (if `target` is missing, krnr will prompt you for it).

---

## Configuration

krnr respects the following environment variables:

| Variable | Description | Default |
|---|---|---|
| `KRNR_HOME` | Directory for the database and logs | `~/.krnr` |
| `KRNR_DB` | Full path to the SQLite database file | `$KRNR_HOME/krnr.db` |
| `EDITOR` | Editor used for `krnr edit` | `vi` (Unix) / `notepad` (Windows) |

---

## Shell Support

krnr intelligently picks the execution environment based on your operating system:

| Platform | Default Shell | Overrides |
|---|---|---|
| **Windows** | `cmd.exe` | `powershell`, `pwsh` |
| **Linux** | `bash` | Any shell on PATH (zsh, fish, etc) |
| **macOS** | `bash` | Any shell on PATH |

Use `--shell <name>` to force a specific executable for a run.

---

## Command Reference

| Command | Description | Example |
|---|---|---|
| `krnr save <name>` | Save a named command set with multiple commands | `krnr save build -c "go build" -c "go test"` |
| `krnr record <name>` | Record commands interactively from your terminal | `krnr record demo` (type `:end` to save) |
| `krnr run <name>` | Execute a saved command set safely | `krnr run deploy --confirm -p env=prod` |
| `krnr list` | List and search saved command sets | `krnr list --filter "deploy" --fuzzy` |
| `krnr describe <name>`| View detailed command set structure and metadata | `krnr describe build` |
| `krnr edit <name>` | Modify a command set using your favorite `$EDITOR` | `krnr edit build` |
| `krnr delete <name>` | Remove a command set from the registry | `krnr delete legacy --yes` |
| `krnr history <name>` | View the versioned history of a command set | `krnr history deploy` |
| `krnr rollback <name>`| Revert a command set to a previous version | `krnr rollback deploy --version 2` |
| `krnr tag <action>` | Manage tags (`add`, `remove`, `list`) for sets | `krnr tag add build production` |
| `krnr export` | Export DB or specific sets to portable SQLite files | `krnr export set build --dst ./build.db` |
| `krnr import` | Import DB or sets with flexible conflict policies | `krnr import set ./build.db --on-conflict merge` |
| `krnr whoami` | Manage your global author identity for recorded runs | `krnr whoami set --name "Alice"` |
| `krnr status` | Check installation and PATH health | `krnr status` |
| `krnr version` | Show current version and build info | `krnr version` |

---

## Roadmap

- [ ] **Secret Encryption**: Encrypt sensitive parameters in the SQLite database.
- [ ] **Remote Sync**: Optional S3/GitHub backup for your registry.
- [ ] **UI Dashboard**: A TUI or Web view for browsing command history and analytics.
- [ ] **Hooks**: Pre-run and post-run hooks for command sets.

---

## Development & Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Recommended: Go 1.25.5+
go build -v -o krnr .
go test ./...
./scripts/lint.sh
```

---

## Troubleshooting

- **PATH issues (Windows)**: If `krnr` is not recognized after installation, ensure you ran `krnr install --add-to-path` in an elevated shell if necessary, or restart your terminal to refresh environment variables.
- **SQLite locks**: Since krnr uses SQLite, avoid manual modification of the `.db` file while a `krnr run` is in progress to prevent database busy errors.
- **Shell not found**: If using `--shell`, ensure the executable is available in your system path.

---

## Security

krnr executes shell commands with the privileges of the current user. 

- Always inspect command sets before running them (use `krnr describe <name>`).
- Use the `--confirm` flag for workflows that perform destructive operations.
- Avoid storing plaintext secrets in command sets; use environment variables and the `env:VAR` parameter syntax instead.

If you find a security vulnerability, please report it privately following our [SECURITY.md](SECURITY.md).

---

## License & Credits

krnr is open-source and licensed under the **MIT License**. See [LICENSE](LICENSE) for details.

Developed for the VoxDroid ecosystem.

- **Discussions**: [GitHub Discussions](https://github.com/VoxDroid/krnr/discussions)
- **Issues**: [Bug reports & feature requests](https://github.com/VoxDroid/krnr/issues)
- **Support**: See [SUPPORT.md](SUPPORT.md) for tiered assistance.
