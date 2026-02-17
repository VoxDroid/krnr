# v1.2.6 - 2026-02-17

This patch fixes a rollback bug that created duplicate version entries, eliminates flaky test failures, and adds Windows cross-compilation support.

## Bug fixes

### Rollback duplicate version entries
Rolling back to a previous version was creating **two** new version records (an "update" + a "rollback") instead of just one. Now only a single "rollback" version entry is created.

### Version preview not updating after a run
After running a command, stale run logs were overriding the left panel — pressing Enter on a version appeared to do nothing, and switching panes didn't update the preview. Run logs are now cleared on pane navigation so version previews and the original detail display correctly.

### Flaky test failures
Fixed intermittent "database is locked" errors when running `go test ./...` — test packages now use isolated temporary databases instead of sharing a single file.

## Portability

### Windows cross-compilation
Unix-only PTY code has been split into build-tagged platform files (`pty_unix.go`, `pty_windows.go`), so `GOOS=windows` builds succeed.

## Upgrade notes
No DB schema changes. Existing version history is unaffected — the fix only prevents future duplicate entries during rollback.
