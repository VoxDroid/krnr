<#
Usage: powershell -File build.ps1 -BuildOutputPath ./build -ProductVersion 1.0.0
This script expects WiX tools (candle.exe/light.exe) to be available on PATH. On CI we can install WiX via Chocolatey.
#>
param(
  [Parameter(Mandatory=$true)] [string] $BuildOutputPath,
  [Parameter(Mandatory=$true)] [string] $ProductVersion
)

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $here

# Ensure build output exists
if (-Not (Test-Path $BuildOutputPath)) {
  Write-Host "Build output path not found: $BuildOutputPath"
  exit 1
}

# Set variables for WiX
$wxs = Join-Path $here "Product.wxs"
# Auto-generate stable GUIDs when placeholders are present so local builds don't require manual edits.
$wxsContent = Get-Content $wxs -Raw
if ($wxsContent -match 'PUT-GUID-HERE') {
  $upg = [guid]::NewGuid().ToString()
  Write-Host "Replacing UpgradeCode placeholder with generated GUID: $upg"
  $wxsContent = $wxsContent -replace 'PUT-GUID-HERE', $upg
}
# Replace any numbered component GUID placeholders (PUT-GUID-0001, PUT-GUID-0002, ...)
$compPlaceholders = [regex]::Matches($wxsContent, 'PUT-GUID-000\d+') | ForEach-Object { $_.Value } | Select-Object -Unique
foreach ($ph in $compPlaceholders) {
  $newguid = [guid]::NewGuid().ToString()
  Write-Host "Replacing component GUID placeholder $ph with generated GUID: $newguid"
  $wxsContent = $wxsContent -replace [regex]::Escape($ph), $newguid
}
$tempWxs = Join-Path $env:TEMP ("krnr_Product_$([guid]::NewGuid().ToString()).wxs")
# Inline simple preprocessor replacements to ensure candle gets concrete values
# Replace ProductVersion directly (WiX requires x.x.x.x format)
$wxsContent = $wxsContent -replace '\$\(var\.ProductVersion\)', $ProductVersion
# Replace BuildOutputPath variable with the absolute path (preserve backslashes)
$absBuildPath = Resolve-Path -Path $BuildOutputPath
$wxsContent = $wxsContent -replace '\$\(var\.BuildOutputPath\)', $absBuildPath.Path
Set-Content -Path $tempWxs -Value $wxsContent -Encoding UTF8
$wxs = $tempWxs

# Provide the license RTF path to candle so the built-in License dialog can pick it up
$licensePath = Join-Path $here 'license.rtf'
# Validate the incoming ProductVersion is a 4-part numeric version accepted by WiX (x.x.x.x)
if (-not ($ProductVersion -match '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$')) {
  Write-Error "ProductVersion '$ProductVersion' is not a valid 4-part numeric version (expected 'x.x.x.x')"
  exit 1
}
$wixobj = Join-Path $here "Product.wixobj"
$outMsi = Join-Path (Resolve-Path "$here\..\..\") "krnr-${ProductVersion}.msi"

Write-Host "Building MSI: $outMsi"

# Compile (.wxs -> .wixobj)
# Prefer explicit WiX install path when present to avoid transient PATH issues.
$candleExe = 'C:\Program Files (x86)\WiX Toolset v3.14\bin\candle.exe'
if (-not (Test-Path $candleExe)) { $candleExe = 'candle.exe' }
& $candleExe -dProductVersion=$ProductVersion -dBuildOutputPath=$BuildOutputPath -dWixUILicenseRtf="$licensePath" -out $wixobj $wxs
if ($LASTEXITCODE -ne 0) { Write-Error "$candleExe failed"; exit $LASTEXITCODE }

# Link (.wixobj -> .msi)
$lightExe = 'C:\Program Files (x86)\WiX Toolset v3.14\bin\light.exe'
if (-not (Test-Path $lightExe)) { $lightExe = 'light.exe' }
& $lightExe -ext WixUIExtension -ext WixUtilExtension -out $outMsi $wixobj
if ($LASTEXITCODE -ne 0) { Write-Error "$lightExe failed"; exit $LASTEXITCODE }

Write-Host "MSI produced at: $outMsi"
Pop-Location
