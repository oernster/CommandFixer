# CommandFixer build script for Windows PowerShell 7
# Usage:
#   .\build.ps1                  Build for Windows (current machine)
#   .\build.ps1 -Test            Run tests before building
#   .\build.ps1 -Coverage        Generate HTML coverage report
#   .\build.ps1 -Race            Run tests with race detector

param(
    [switch]$Test,
    [switch]$Coverage,
    [switch]$Race,
    [switch]$Clean
)

$ErrorActionPreference = 'Stop'

$BinaryName = "commandfixer.exe"
$CoverFile  = "coverage.out"
$CoverHTML  = "coverage.html"

# ---- Clean ---------------------------------------------------------------

if ($Clean) {
    Write-Host "Cleaning..." -ForegroundColor Cyan
    Remove-Item -Force -ErrorAction SilentlyContinue $BinaryName, $CoverFile, $CoverHTML
    Write-Host "  Done." -ForegroundColor Green
    exit 0
}

# ---- Tests ---------------------------------------------------------------

if ($Race) {
    Write-Host "Running tests with race detector..." -ForegroundColor Cyan
    go test -race ./...
    if ($LASTEXITCODE -ne 0) { Write-Host "Tests failed." -ForegroundColor Red; exit 1 }
    Write-Host "  All tests passed." -ForegroundColor Green
    exit 0
}

if ($Coverage) {
    Write-Host "Running tests with coverage..." -ForegroundColor Cyan
    go test -coverprofile=$CoverFile -covermode=atomic ./...
    if ($LASTEXITCODE -ne 0) { Write-Host "Tests failed." -ForegroundColor Red; exit 1 }
    go tool cover -func=$CoverFile
    go tool cover -html=$CoverFile -o $CoverHTML
    Write-Host ""
    Write-Host "Coverage report written to: $CoverHTML" -ForegroundColor Green
    Start-Process $CoverHTML
    exit 0
}

if ($Test) {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test ./...
    if ($LASTEXITCODE -ne 0) { Write-Host "Tests failed." -ForegroundColor Red; exit 1 }
    Write-Host "  All tests passed." -ForegroundColor Green
}

# ---- Build ---------------------------------------------------------------

Write-Host "Building $BinaryName..." -ForegroundColor Cyan
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o $BinaryName .
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed." -ForegroundColor Red; exit 1 }

$size = (Get-Item $BinaryName).Length / 1KB
Write-Host "  Built: $BinaryName ($([math]::Round($size, 1)) KB)" -ForegroundColor Green
