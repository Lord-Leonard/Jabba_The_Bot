param(
  [string]$Source = "web/dist",
  [string]$Target = "internal/adapters/inbound/http/ui"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $Source)) {
  throw "Source folder '$Source' does not exist. Build frontend first (cd web; npm run build)."
}

if (Test-Path $Target) {
  Remove-Item -Recurse -Force $Target
}

New-Item -ItemType Directory -Force $Target | Out-Null
Copy-Item -Path (Join-Path $Source "*") -Destination $Target -Recurse -Force

Write-Host "Embedded UI assets synced from '$Source' to '$Target'"
