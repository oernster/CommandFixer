# CommandFixer uninstaller for Windows PowerShell 5 (powershell.exe) and PowerShell 7 (pwsh)
# Run with: .\uninstall.ps1
# Or from any dir: powershell -ExecutionPolicy Bypass -File .\uninstall.ps1

param(
    [string]$InstallDir,
    [string]$ConfigDir,
    [switch]$RemoveConfig
)

$ErrorActionPreference = 'Stop'

# The locations, the profile paths and the hook markers live in one place,
# shared with install.ps1. The manual fallback below depends on the markers
# being exactly what the binary wrote, so it reads them rather than restating
# them. A param default cannot read this (defaults are evaluated before the
# body runs), so the fallback is applied here instead.
. (Join-Path $PSScriptRoot 'profile-hook.ps1')

if (-not $InstallDir) { $InstallDir = $CommandFixerInstallDir }
if (-not $ConfigDir)  { $ConfigDir  = $CommandFixerConfigDir }

$BinaryName = $CommandFixerBinaryName
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

    # Remove the hook block from both profiles by hand, because the binary that
    # would normally do it is gone. The paths and the markers come from
    # profile-hook.ps1, so this cannot drift away from what was written.
    $snippetStart = $CommandFixerSnippetStart
    $snippetEnd   = $CommandFixerSnippetEnd

    # Not $profile: that is an automatic PowerShell variable and shadowing it
    # inside this scope is a trap for anything added below.
    foreach ($profilePath in (Get-CommandFixerProfilePaths)) {
        if (-not (Test-Path $profilePath)) { continue }
        $content = Get-Content $profilePath -Raw
        if (-not $content.Contains($snippetStart)) {
            Write-Host "  Not installed in: $profilePath" -ForegroundColor Gray
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
        Set-Content $profilePath -Value $content -NoNewline
        Write-Host "  Removed hook from: $profilePath" -ForegroundColor Green
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
