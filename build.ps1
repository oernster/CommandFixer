# CommandFixer build, test, lint and coverage script for Windows PowerShell 7.
#
# This is the whole workflow. It replaced a Makefile that duplicated it on a
# platform this tool's developer does not use: CommandFixer corrects PowerShell
# commands, its users are on Windows, and a checklist you cannot run is a
# checklist that goes stale. One runner, runnable where the work happens.
#
# Usage:
#   .\build.ps1                  Build for Windows (current machine)
#   .\build.ps1 -Test            Run tests, then build
#   .\build.ps1 -Lint            gofmt, go vet and staticcheck
#   .\build.ps1 -Coverage        Coverage report, failing below the floor
#   .\build.ps1 -Clean           Remove build artefacts
#
# There is deliberately no race-detector switch: Go's race detector requires
# cgo, this tool is deliberately CGO-free and Windows machines here have no C
# toolchain, so the switch could only ever fail with "-race requires cgo". A
# command that cannot run is worse than an absent one, because it reads as a
# check that is being performed.

# CmdletBinding so an unrecognised switch is an error rather than silence.
# Without it PowerShell drops unknown named arguments into $args, so a stale
# `-Race` or a mistyped `-Covrage` ran a plain build and reported success: the
# check you asked for never ran and nothing said so.
[CmdletBinding()]
param(
    [switch]$Test,
    [switch]$Coverage,
    [switch]$Lint,
    [switch]$Clean
)

$ErrorActionPreference = 'Stop'

$BinaryName = "commandfixer.exe"
$CoverFile  = "coverage.out"
$CoverHTML  = "coverage.html"
$VersionFile = "VERSION"

# Printed when VERSION cannot be read, and compiled in as the default by
# main.go for the same reason: a binary that was built without the version
# should say so rather than claim a number it does not have.
$FallbackVersion = "0.0.0-dev"

# The floor the suite already clears, not an aspiration. Measured 2026-08-06 at
# 83.5% total (corrector 100%, config 92.1%, logger 91.4%, shell 87.7%, main
# 65.3%). Raise it when the suite earns it; never lower it to make a run pass.
$CoverageFloor = 83.0

# staticcheck is a strict superset of go vet. Run from source so nothing has to
# be installed and kept up to date.
$StaticcheckPackage = "honnef.co/go/tools/cmd/staticcheck@latest"

function Get-BuildVersion {
    # The VERSION file is the single source of truth. Nothing else in the
    # repository holds a real version string.
    if (Test-Path $VersionFile) {
        $version = (Get-Content $VersionFile -Raw).Trim()
        if ($version) { return $version }
    }
    Write-Host "  No VERSION file; building as $FallbackVersion." -ForegroundColor Yellow
    return $FallbackVersion
}

function Assert-Succeeded([string]$What) {
    if ($LASTEXITCODE -ne 0) {
        Write-Host "$What failed." -ForegroundColor Red
        exit 1
    }
}

# ---- Clean ---------------------------------------------------------------

if ($Clean) {
    Write-Host "Cleaning..." -ForegroundColor Cyan
    Remove-Item -Force -ErrorAction SilentlyContinue $BinaryName, $CoverFile, $CoverHTML
    Write-Host "  Done." -ForegroundColor Green
    exit 0
}

# ---- Lint ----------------------------------------------------------------

if ($Lint) {
    Write-Host "Checking formatting..." -ForegroundColor Cyan
    # gofmt reports unformatted files on stdout and still exits 0, so the
    # output is the result rather than the exit code.
    $unformatted = & gofmt -l .
    if ($unformatted) {
        Write-Host "  Not gofmt-clean:" -ForegroundColor Red
        $unformatted | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
        exit 1
    }
    Write-Host "  gofmt clean." -ForegroundColor Green

    Write-Host "Running go vet..." -ForegroundColor Cyan
    go vet ./...
    Assert-Succeeded "go vet"
    Write-Host "  go vet clean." -ForegroundColor Green

    Write-Host "Running staticcheck..." -ForegroundColor Cyan
    go run $StaticcheckPackage ./...
    Assert-Succeeded "staticcheck"
    Write-Host "  staticcheck clean." -ForegroundColor Green
    exit 0
}

# ---- Tests ---------------------------------------------------------------

if ($Coverage) {
    Write-Host "Running tests with coverage..." -ForegroundColor Cyan
    # The go flags are quoted deliberately. Unquoted, PowerShell splits
    # -coverprofile=coverage.out at the dot and go is handed ".out" as a
    # package, which fails and leaves a truncated file called "coverage"
    # behind. Quoting passes each flag through as one argument.
    go test "-coverprofile=$CoverFile" "-covermode=atomic" ./...
    Assert-Succeeded "Tests"

    $summary = & go tool cover "-func=$CoverFile"
    $summary | Write-Host

    # The total is the last line: "total:  (statements)  83.5%".
    $totalLine = $summary | Select-String 'total:' | Select-Object -Last 1
    if (-not $totalLine) {
        Write-Host "Could not read a coverage total from the profile." -ForegroundColor Red
        exit 1
    }
    if ($totalLine -notmatch '([0-9]+(?:\.[0-9]+)?)%') {
        Write-Host "Could not parse the coverage total: $totalLine" -ForegroundColor Red
        exit 1
    }
    $total = [double]$Matches[1]

    go tool cover "-html=$CoverFile" "-o" $CoverHTML
    Write-Host ""
    Write-Host "Coverage report written to: $CoverHTML" -ForegroundColor Green

    if ($total -lt $CoverageFloor) {
        Write-Host ("Coverage {0}% is below the floor of {1}%." -f $total, $CoverageFloor) -ForegroundColor Red
        exit 1
    }
    Write-Host ("Coverage {0}%, floor {1}%." -f $total, $CoverageFloor) -ForegroundColor Green
    exit 0
}

if ($Test) {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test ./...
    Assert-Succeeded "Tests"
    Write-Host "  All tests passed." -ForegroundColor Green
}

# ---- Build ---------------------------------------------------------------

$version = Get-BuildVersion
Write-Host "Building $BinaryName ($version)..." -ForegroundColor Cyan
$env:GOOS   = "windows"
$env:GOARCH = "amd64"
go build "-ldflags=-s -w -X main.appVersion=$version" -o $BinaryName .
Assert-Succeeded "Build"

$size = (Get-Item $BinaryName).Length / 1KB
Write-Host "  Built: $BinaryName ($([math]::Round($size, 1)) KB)" -ForegroundColor Green
