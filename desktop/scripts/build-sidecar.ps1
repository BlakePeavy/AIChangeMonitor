# Build the Go aichange engine as a Tauri sidecar (Windows).
# Writes src-tauri/binaries/aichange-<target-triple>.exe
# Does not replace the repo-root aichange.exe CLI.
$ErrorActionPreference = "Stop"
$desktop = Split-Path -Parent $PSScriptRoot
$goMod = Split-Path -Parent $desktop
$binDir = Join-Path $desktop "src-tauri\binaries"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null

$triple = $env:TAURI_ENV_TARGET_TRIPLE
if (-not $triple) { $triple = $env:TARGET }
if (-not $triple) { $triple = (rustc --print host-tuple).Trim() }
if (-not $triple) { throw "could not determine rustc target triple" }

$out = Join-Path $binDir "aichange-$triple.exe"
$env:CGO_ENABLED = "0"
if ($triple -match "windows") { $env:GOOS = "windows" }
if ($triple -match "aarch64|arm64") { $env:GOARCH = "arm64" }
elseif ($triple -match "x86_64|amd64") { $env:GOARCH = "amd64" }

Push-Location $goMod
try {
  Write-Host "go build -o $out ."
  go build -o $out .
} finally {
  Pop-Location
}
Write-Host $out
