# KRNR Demo Commands — Windows (PowerShell & cmd)

> Safety: these commands use an isolated `KRNR_HOME` and are safe to paste and run as-is. Do NOT run them without the `KRNR_HOME` setup if you want to preserve your primary registry.

## PowerShell — copy/paste entire block

```powershell
# 1. Setup isolated demo environment
$env:KRNR_HOME = Join-Path $env:TEMP 'krnr_demo'
Remove-Item -Recurse -Force $env:KRNR_HOME -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $env:KRNR_HOME | Out-Null
Write-Host "KRNR_HOME = $env:KRNR_HOME"

# 2. whoami
krnr whoami set --name "Demo User" --email demo@example.com
krnr whoami show

# 3. Save demo set
krnr save demo -d "Demo set" -c "echo Hello from demo" -c "echo Demo done"
krnr describe demo
krnr list

# 4. Dry-run (shows redaction) and run
krnr run demo --dry-run --verbose
# To actually execute (confirm prompt): printf "y\n" | krnr run demo --confirm

# 5. Parameterized run with env-sourced token
krnr save with-param -d "Param demo" -c "echo User: {{user}}" -c "echo Token: {{token}}"
$env:API_TOKEN = 's3cr3t'
krnr run with-param --param user=demo --param token=env:API_TOKEN --dry-run --verbose

# 6. Tagging & discovery
krnr tag add demo utils
krnr tag list demo
krnr list --tag utils
krnr save project-build -d "Build demo" -c "echo building project"
krnr list --filter build
krnr list --filter bld --fuzzy

# 7. Scripted record (here-string)
@"
echo recorded-first
echo recorded-second
:end
"@ | krnr record scripted
krnr describe scripted
krnr run scripted --dry-run

# 8. Edit, history, rollback (v1 example)
krnr edit demo -c "echo Updated demo" -c "echo More updates"
krnr history demo
krnr rollback demo --version 1
krnr describe demo

# 9. Delete non-interactive
krnr delete project-build --yes || Write-Host "project-build may not exist"

# 10. Install/uninstall preview
krnr install --user --from ./krnr --dry-run
krnr uninstall --dry-run
krnr status

# 11. Export test that produces an exported DB (test helper)
go test ./cmd -run TestExportDatabase -v || Write-Host "Export test may require test environment"

# 12. Cleanup
Remove-Item -Recurse -Force $env:KRNR_HOME -ErrorAction SilentlyContinue
Write-Host "PowerShell demo completed and cleaned up."
```

## Bash — copy/paste entire block

```bash
# 1. Setup isolated demo environment
export KRNR_HOME="$(mktemp -d -t krnr_demo_XXXX)"
echo "KRNR_HOME=$KRNR_HOME"

# 2. whoami
krnr whoami set --name "Demo User" --email demo@example.com
krnr whoami show

# 3. Save demo set
krnr save demo -d "Demo set" -c "echo Hello from demo" -c "echo Demo done"
krnr describe demo
krnr list

# 4. Dry-run (shows redaction) and run
krnr run demo --dry-run --verbose
# To actually execute (confirm prompt): printf "y\n" | krnr run demo --confirm

# 5. Parameterized run with env-sourced token
krnr save with-param -d "Param demo" -c "echo User: {{user}}" -c "echo Token: {{token}}"
export API_TOKEN='s3cr3t'
krnr run with-param --param user=demo --param token=env:API_TOKEN --dry-run --verbose

# 6. Tagging & discovery
krnr tag add demo utils
krnr tag list demo
krnr list --tag utils
krnr save project-build -d "Build demo" -c "echo building project"
krnr list --filter build
krnr list --filter bld --fuzzy

# 7. Scripted record (here-doc)
cat <<'EOF' | krnr record scripted
echo recorded-first
echo recorded-second
:end
EOF
krnr describe scripted
krnr run scripted --dry-run

# 8. Edit, history, rollback (v1 example)
krnr edit demo -c "echo Updated demo" -c "echo More updates"
krnr history demo
krnr rollback demo --version 1
krnr describe demo

# 9. Delete non-interactive
krnr delete project-build --yes || echo "project-build may not exist"

# 10. Install/uninstall preview
krnr install --user --from ./krnr --dry-run
krnr uninstall --dry-run
krnr status || true

# 11. Export test that produces an exported DB (test helper)
go test ./cmd -run TestExportDatabase -v || echo "Export test may require test environment"

# 12. Cleanup
rm -rf "$KRNR_HOME"
echo "Bash demo completed and cleaned up."
```

```powershell
# 1) Setup isolated environment
export KRNR_HOME="$(mktemp -d -t krnr_demo_XXXX)"
echo "KRNR_HOME=$KRNR_HOME"

# 2) Set a default author
krnr whoami set --name "Demo User" --email demo@example.com
krnr whoami show

# 3) Save a basic command set and verify
krnr save demo -d "Demo set" -c "echo Hello from demo" -c "echo Demo done"
krnr describe demo
krnr list

# 4) Dry-run & actual run
krnr run demo --dry-run --verbose
# To run interactively: printf "y\n" | krnr run demo --confirm

# 5) Parameterized run with env sourcing & redaction
krnr save greet -d "Greeting" -c "echo Welcome {{user}}" -c "echo Token: {{token}}"
export API_TOKEN='s3cr3t'
krnr run greet --param user=demo --param token=env:API_TOKEN --dry-run --verbose

# 6) Tagging, discovery & fuzzy
krnr tag add demo utils
krnr tag list demo
krnr list --tag utils
krnr save project-build -d "Build set" -c "echo building project"
krnr list --filter bld --fuzzy

# 7) Scripted record (here-doc with :end sentinel)
cat <<'EOF' | krnr record scripted
echo recorded-first
echo recorded-second
:end
EOF
krnr describe scripted
krnr run scripted --dry-run

# 8) Edit non-interactive, history, rollback
krnr edit demo -c "echo Updated demo" -c "echo Updated more"
krnr history demo
# Example rollback: pick a version number from history, e.g. 1
# krnr rollback demo --version 1
krnr describe demo

# 9) Delete (non-interactive)
krnr delete project-build --yes || echo "project-build may not exist"

# 10) Install/uninstall preview and status
krnr install --user --from ./krnr --dry-run
krnr uninstall --dry-run
krnr status || true

# 11) Export/Import via test helper
go test ./cmd -run TestExportDatabase -v || echo "Export test may require test environment"

# 12) Cleanup
rm -rf "$KRNR_HOME"
echo "Demo completed and cleaned up."
```

---

## Quick testing tips & checks
- Verify a run produced expected output (POSIX):
  - `krnr run demo --dry-run --verbose | grep -i "hello from demo" || echo "dry-run output missing"`
- On PowerShell, check a command output contains expected text after run:
  - `krnr run demo | Out-String -Stream | Select-String "Demo done"`

---

Generated on 2026-01-11 — ready to use demo sequences for CI or local testing.
