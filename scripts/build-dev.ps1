# Build a local development binary as logyDEV.exe (does not overwrite release logy).
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$Version = if ($env:LOGY_DEV_VERSION) { $env:LOGY_DEV_VERSION } else { '0.0.0-dev' }
$Out = Join-Path $Root 'logyDEV.exe'
$Ldflags = "-X logy/internal/version.Version=$Version"

Write-Host "Building $Out (version $Version)..."
go build -ldflags $Ldflags -o $Out ./cmd/logy
& $Out version
Write-Host ""
Write-Host "Run with: .\logyDEV.exe <command>"
