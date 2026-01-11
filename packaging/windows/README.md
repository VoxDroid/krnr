This folder contains templates and helper scripts for creating Windows packaging artifacts (MSI via WiX, winget manifest, Scoop/Scoop JSON, Chocolatey nuspec).

Quick start (local, Windows):
- Install WiX Toolset (choco install wixtoolset)
- Build the Go binary: `go build -o build\\krnr.exe .`
- Run the MSI builder: `powershell -File packaging/windows/wix/build.ps1 -BuildOutputPath .\\build -ProductVersion 0.1.0`

CI notes:
- `/.github/workflows/build-msi-windows.yml` performs the build on `windows-latest`, produces an MSI and uploads it to GitHub Releases. The release job was extended in `release.yml` to build the MSI on release tags and upload it to the release.
- The workflow computes SHA256 for the MSI; the `packaging/windows/winget/krnr.yml` file is a manifest template — CI should replace the `Url` and `Sha256` values before submitting to winget-pkgs. We plan to automate manifest updates and PR creation on release.

Signing & packaging notes:
- These templates produce unsigned MSI. For production, add an Authenticode signing step in CI using `signtool.exe` and a PFX stored in secrets.
- The WiX build embeds the license (license.rtf), generates stable GUIDs when placeholders are present, and inlines `ProductVersion` and `BuildOutputPath` so the MSI is self-contained (single-file install with embedded CAB by default).
