# CommandFixer uninstaller for Windows PowerShell 5 (powershell.exe) and PowerShell 7 (pwsh)
# Run with: .\uninstall.ps1
# Or from any dir: powershell -ExecutionPolicy Bypass -File .\uninstall.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\CommandFixer",
    [string]$ConfigDir  = "$env:USERPROFILE\.typo-fixer",
    [switch]$RemoveConfig
)

$ErrorActionPreference = 'Stop'

$BinaryName = "commandfixer.exe"
$BinaryPath = Join-Path $InstallDir $BinaryName

# ---- 1. Remove PowerShell profile hooks (PS5 + PS7) -------------------------

Write-Host ""
Write-Host "Removing PowerShell profile hooks (PS5 + PS7)..." -ForegroundColor Cyan

if (Test-Path $BinaryPath) {
    & $BinaryPath uninstall
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Profile hook removal reported exit $LASTEXITCODE (hooks may already be absent)." -ForegroundColor Yellow
    }
} else {
    Write-Host "Binary not found at: $BinaryPath" -ForegroundColor Yellow
    Write-Host "Attempting manual profile cleanup..." -ForegroundColor Cyan

    # Remove hook block from both profiles manually if binary is gone
    $profiles = @(
        (Join-Path $HOME "Documents\PowerShell\profile.ps1"),
        (Join-Path $HOME "Documents\WindowsPowerShell\profile.ps1")
    )
    $snippetStart = "# CommandFixer Integration - DO NOT EDIT"
    $snippetEnd   = "# End CommandFixer Integration"

    foreach ($profile in $profiles) {
        if (-not (Test-Path $profile)) { continue }
        $content = Get-Content $profile -Raw
        if (-not $content.Contains($snippetStart)) {
            Write-Host "  Not installed in: $profile" -ForegroundColor Gray
            continue
        }
        $startIdx = $content.IndexOf($snippetStart)
        $endIdx   = $content.IndexOf($snippetEnd, $startIdx)
        if ($endIdx -ge 0) {
            $endIdx += $snippetEnd.Length
            if ($endIdx -lt $content.Length -and $content[$endIdx] -eq "`n") { $endIdx++ }
            $content = $content.Substring(0, $startIdx).TrimEnd("`n") + $content.Substring($endIdx)
        } else {
            $content = $content.Substring(0, $startIdx).TrimEnd("`n") + "`n"
        }
        Set-Content $profile -Value $content -NoNewline
        Write-Host "  Removed hook from: $profile" -ForegroundColor Green
    }
}

# ---- 2. Remove binary and install directory ----------------------------------

if (Test-Path $InstallDir) {
    Remove-Item -Recurse -Force $InstallDir
    Write-Host "Removed install directory: $InstallDir" -ForegroundColor Green
} else {
    Write-Host "Install directory not found (already removed): $InstallDir" -ForegroundColor Gray
}

# ---- 3. Remove install dir from user PATH ------------------------------------

$userPath = [System.Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -like "*$InstallDir*") {
    $newPath = ($userPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
    [System.Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "Removed $InstallDir from user PATH." -ForegroundColor Green
} else {
    Write-Host "$InstallDir not in PATH (already removed)." -ForegroundColor Gray
}

# ---- 4. Optionally remove config and log ------------------------------------

if ($RemoveConfig) {
    if (Test-Path $ConfigDir) {
        Remove-Item -Recurse -Force $ConfigDir
        Write-Host "Removed config directory: $ConfigDir" -ForegroundColor Green
    } else {
        Write-Host "Config directory not found: $ConfigDir" -ForegroundColor Gray
    }
} else {
    Write-Host ""
    Write-Host "Config and log kept at: $ConfigDir" -ForegroundColor Gray
    Write-Host "  To remove them too: .\uninstall.ps1 -RemoveConfig"
}

# ---- 5. Done -----------------------------------------------------------------

Write-Host ""
Write-Host "CommandFixer uninstalled." -ForegroundColor Green
Write-Host "Restart PowerShell to complete removal from both powershell.exe and pwsh."
Write-Host ""
