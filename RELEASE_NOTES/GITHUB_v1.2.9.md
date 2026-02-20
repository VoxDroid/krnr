# v1.2.9 - 2026-02-20

This patch corrects a TUI/status reporting inaccuracy: previously a system directory appearing on PATH could cause `krnr status` (and the TUI) to report `on PATH: true` for a system install even when the `krnr` binary was not present in that directory.

**Fix:** `krnr status` now only reports `on PATH:true` when the `krnr` binary is actually present at the expected user/system location (or when `exec.LookPath` resolves the executable). The change prevents false-positive status messages in the TUI and CLI.

No DB or CLI-breaking changes. Users should upgrade to v1.2.9 to get more accurate installer/status diagnostics.