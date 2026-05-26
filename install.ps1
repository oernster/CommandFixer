# CommandFixer installer for Windows PowerShell 5 (powershell.exe) and PowerShell 7 (pwsh)
# Run with: .\install.ps1
# Or from any dir: powershell -ExecutionPolicy Bypass -File .\install.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\CommandFixer",
    [string]$ConfigDir  = "$env:USERPROFILE\.typo-fixer"
)

$ErrorActionPreference = 'Stop'

$BinaryName    = "commandfixer.exe"
$BinarySource  = Join-Path $PSScriptRoot $BinaryName
$BinaryDest    = Join-Path $InstallDir $BinaryName
$ConfigDest    = Join-Path $ConfigDir "config.json"
$ConfigExample = Join-Path $PSScriptRoot "config.example.json"

# ---- 1. Locate the binary -----------------------------------------------

if (-not (Test-Path $BinarySource)) {
    Write-Host ""
    Write-Host "Binary not found at: $BinarySource" -ForegroundColor Red
    Write-Host ""
    Write-Host "Build it first:"
    Write-Host "  go build -o commandfixer.exe ."
    Write-Host ""
    exit 1
}

# ---- 2. Copy binary to install dir --------------------------------------

Write-Host "Installing $BinaryName to: $BinaryDest" -ForegroundColor Cyan
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path $BinarySource -Destination $BinaryDest -Force
Write-Host "  Done." -ForegroundColor Green

# ---- 3. Add install dir to user PATH if not already there ---------------

$userPath = [System.Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable(
        'Path',
        "$userPath;$InstallDir",
        'User'
    )
    Write-Host "Added $InstallDir to user PATH." -ForegroundColor Cyan
    Write-Host "  You may need to restart your terminal for PATH to take effect."
} else {
    Write-Host "$InstallDir already in PATH." -ForegroundColor Gray
}

# ---- 4. Copy example config if none exists ------------------------------

if (-not (Test-Path $ConfigDest)) {
    New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    Copy-Item -Path $ConfigExample -Destination $ConfigDest -Force
    Write-Host "Copied example config to: $ConfigDest" -ForegroundColor Cyan
    Write-Host "  Edit it to add your own typo corrections."
} else {
    Write-Host "Config already exists at: $ConfigDest" -ForegroundColor Gray
}

# ---- 5. Install PowerShell profile hook ---------------------------------

Write-Host ""
Write-Host "Installing PowerShell profile hook (PS5 + PS7)..." -ForegroundColor Cyan
& $BinaryDest install
if ($LASTEXITCODE -ne 0) {
    Write-Host "Profile hook installation failed (exit $LASTEXITCODE)." -ForegroundColor Yellow
    Write-Host "Run manually: $BinaryDest install"
}

# ---- 6. Done ------------------------------------------------------------

Write-Host ""
Write-Host "CommandFixer installed successfully (PowerShell 5 + 7)." -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Edit config: $ConfigDest"
Write-Host "  2. Restart PowerShell (both powershell.exe and pwsh will work)"
Write-Host "  3. Type a typo and press Enter - it gets corrected automatically"
Write-Host ""
Write-Host "Other commands:"
Write-Host "  commandfixer stats      - show correction history"
Write-Host "  commandfixer uninstall  - remove the profile hook from all shells"
Write-Host "  commandfixer help       - show all commands"
