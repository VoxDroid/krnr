This is the level expected for **serious OSS, senior portfolios, or internal tooling at companies**.

---

# Project Overview — **krnr**

**Kernel Runner**

> **krnr** is a cross-platform CLI tool that provides a global, persistent command registry backed by SQLite, enabling users to save, name, execute, export, and share terminal workflows without writing scripts or managing per-directory configuration files.

---

## 1. Purpose & Vision

### 1.1 Problem

Terminal users repeatedly execute complex workflows but rely on:

* shell scripts (file overhead)
* aliases (non-portable, invisible)
* task runners (project-scoped, heavy)
* notes or copy-paste (error-prone)

These approaches fail across **directories, machines, shells, and operating systems**.

---

### 1.2 Solution

**krnr** introduces a **global execution layer**:

* workflows are **stored centrally**
* commands are **named and discoverable**
* execution is **explicit and safe**
* storage is **structured, queryable, and portable**

SQLite provides durability, indexing, versioning, and future extensibility.

---

## 2. Core Design Principles

| Principle            | Description                            |
| -------------------- | -------------------------------------- |
| Global-first         | Commands accessible from any directory |
| Scriptless           | No `.sh`, `.ps1`, `.bat` files         |
| Cross-platform       | Windows, Linux, macOS                  |
| Explicit execution   | No implicit or background runs         |
| Structured storage   | SQLite, not flat files                 |
| Safe defaults        | Stop-on-error, confirmations           |
| Professional tooling | Linting, formatting, hooks             |

---

## 3. Technology Stack

| Layer         | Technology                                              |
| ------------- | ------------------------------------------------------- |
| Language      | Go (≥ 1.25.5)                                             |
| CLI Framework | Cobra                                                   |
| Database      | SQLite (via `modernc.org/sqlite` or `mattn/go-sqlite3`) |
| Config        | Dot-directory (`~/.krnr/`)                              |
| Execution     | OS-native shells                                        |
| Linting       | golangci-lint                                           |
| Formatting    | gofmt, goimports                                        |
| Hooks         | pre-commit                                              |
| CI (optional) | GitHub Actions                                          |

---

## 4. High-Level Architecture

```
┌──────────────────────┐
│ CLI Layer (cobra)    │
│  krnr <command>      │
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ Registry Service     │
│  SQLite-backed       │
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ Execution Engine     │
│  OS-aware shells     │
└──────────────────────┘
```

---

## 5. Repository File Structure (Final)

```
krnr/
├── cmd/
│   ├── root.go
│   ├── save.go
│   ├── run.go
│   ├── list.go
│   ├── describe.go
│   ├── edit.go
│   ├── delete.go
│   ├── export.go
│   └── import.go
│
├── internal/
│   ├── db/
│   │   ├── db.go              # DB initialization & migrations
│   │   ├── schema.sql         # SQLite schema
│   │   └── migrations.go
│   │
│   ├── registry/
│   │   ├── registry.go        # CRUD operations
│   │   └── models.go
│   │
│   ├── executor/
│   │   ├── executor.go
│   │   ├── executor_windows.go
│   │   └── executor_unix.go
│   │
│   ├── recorder/
│   │   └── recorder.go
│   │
│   ├── exporter/
│   │   └── sqlite_export.go
│   │
│   ├── importer/
│   │   └── sqlite_import.go
│   │
│   ├── config/
│   │   └── paths.go
│   │
│   └── utils/
│       ├── editor.go
│       ├── confirm.go
│       └── time.go
│
├── .golangci.yml
├── .pre-commit-config.yaml
├── docs/
│   ├── architecture.md
│   ├── database.md
│   └── roadmap.md
├── scripts/
│   ├── lint.sh
│   ├── fmt.sh
│   └── build.sh
├── tests/
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 6. Database Design (SQLite)

### 6.1 Database Location

| OS      | Path                            |
| ------- | ------------------------------- |
| Windows | `C:\Users\<user>\.krnr\krnr.db` |
| Linux   | `~/.krnr/krnr.db`               |
| macOS   | `~/.krnr/krnr.db`               |

---

### 6.2 Schema

```sql
CREATE TABLE command_sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL,
    last_run DATETIME
);

CREATE TABLE commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command_set_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    command TEXT NOT NULL,
    FOREIGN KEY(command_set_id) REFERENCES command_sets(id)
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE command_set_tags (
    command_set_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (command_set_id, tag_id)
);
```

---

## 7. Command Execution Engine

### 7.1 Shell Strategy

| OS            | Shell      | Invocation            |
| ------------- | ---------- | --------------------- |
| Windows       | cmd.exe    | `cmd /C <cmd>`        |
| Windows (alt) | PowerShell | `pwsh -Command <cmd>` |
| Linux         | bash       | `bash -c <cmd>`       |
| macOS         | bash       | `bash -c <cmd>`       |

Detected via:

```go
runtime.GOOS
```

---

### 7.2 Execution Behavior

| Aspect  | Behavior                  |
| ------- | ------------------------- |
| Order   | Sequential                |
| CWD     | Current directory         |
| Output  | Stdout/stderr passthrough |
| Failure | Stop on error (default)   |
| Flags   | dry-run, confirm, verbose |

---

## 8. CLI Commands Overview

| Command                | Description                   |
| ---------------------- | ----------------------------- |
| `krnr save <name>`     | Record workflow interactively |
| `krnr <name>`          | Execute workflow              |
| `krnr list`            | List workflows                |
| `krnr describe <name>` | Show details                  |
| `krnr edit <name>`     | Open workflow in editor       |
| `krnr delete <name>`   | Remove workflow               |
| `krnr export`          | Export workflows              |
| `krnr import`          | Import workflows              |

---

## 9. Linting & Formatting Strategy (Professional)

### 9.1 Formatting

Mandatory:

* `gofmt`
* `goimports`

Enforced via:

```bash
go fmt ./...
```

---

### 9.2 Linting (golangci-lint)

`.golangci.yml` (example):

```yaml
run:
  timeout: 3m

linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - ineffassign
    - unused
    - revive
    - gosimple

issues:
  exclude-use-default: false
```

---

## 10. Pre-Commit Hooks

### 10.1 Tool

Uses **pre-commit** framework (language-agnostic, widely adopted).

### 10.2 `.pre-commit-config.yaml`

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer

  - repo: https://github.com/golangci/golangci-lint
    rev: v2.8.0
    hooks:
      - id: golangci-lint

  - repo: local
    hooks:
      - id: gofmt
        name: gofmt
        entry: gofmt -w .
        language: system
        files: \.go$
```

### 10.3 Developer Flow

```bash
pip install pre-commit
pre-commit install
```

Now:

* commits fail if code is unformatted
* lint violations are blocked
* quality is enforced automatically

---

## 11. Build & Distribution

| Platform | Output     |
| -------- | ---------- |
| Windows  | `krnr.exe` |
| Linux    | `krnr`     |
| macOS    | `krnr`     |

Cross-compilation supported via Go.

---

## 12. Security & Safety

| Area                 | Policy          |
| -------------------- | --------------- |
| Background execution | ❌ Disabled      |
| Auto-run             | ❌ Not allowed   |
| Network access       | ❌ None          |
| Secrets handling     | ❌ Not in v1     |
| Permissions          | User-level only |

---

## 13. Why SQLite Is the Right Choice

| Reason             | Benefit                    |
| ------------------ | -------------------------- |
| Structured storage | No schema drift            |
| Queryable          | Fast filtering & search    |
| Atomic             | No corruption risk         |
| Portable           | Single file                |
| Extensible         | Parameters, history, stats |

---

## 14. Final Positioning Statement

> **krnr is a global, SQLite-backed command execution layer that lets users save, name, run, and share terminal workflows across Windows, Linux, and macOS without scripts or shell-specific configuration.**

---
