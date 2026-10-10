# Runs the checks required before opening a PR. CI runs the same script.
# Usage: ./tools/check.ps1

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $PSScriptRoot)

$failed = @()

Write-Host '== gofmt -l .'
$unformatted = gofmt -l .
if ($LASTEXITCODE -ne 0) { $failed += 'gofmt' }
elseif ($unformatted) {
    $unformatted | ForEach-Object { Write-Host "  $_" }
    $failed += 'gofmt'
}

Write-Host '== go mod tidy -diff'
go mod tidy -diff
if ($LASTEXITCODE -ne 0) { $failed += 'go mod tidy' }

Write-Host '== go vet ./...'
go vet ./...
if ($LASTEXITCODE -ne 0) { $failed += 'go vet' }

Write-Host '== go test ./...'
go test ./...
if ($LASTEXITCODE -ne 0) { $failed += 'go test' }

if ($failed.Count -gt 0) {
    Write-Host "FAILED: $($failed -join ', ')"
    exit 1
}
Write-Host 'All checks passed.'
