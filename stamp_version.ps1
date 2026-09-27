# Writes the version held in VERSION into every stamped static file.
#
# Markdown and the GitHub Pages site cannot read VERSION when they are shown, so
# each place that names the version carries a token instead:
#
#     <!--VERSION-->MAJOR.MINOR.PATCH<!--/VERSION-->
#
# This rewrites whatever sits between the two markers, across the root *.md
# files and everything under docs/. It is idempotent: a file that already
# carries the current version is not written at all. build.ps1 runs it before
# every build, so a release cannot ship a site naming the previous version,
# which is exactly what happened when 1.4.0 was first committed.
#
# Usage, from the repository root:
#
#     .\stamp_version.ps1

$ErrorActionPreference = 'Stop'

$versionFile = Join-Path $PSScriptRoot 'VERSION'
$docsDir     = Join-Path $PSScriptRoot 'docs'

# A release version is MAJOR.MINOR.PATCH. Anything else is refused rather than
# written into the site, where it would read as a real release.
$versionShape = '^\d+\.\d+\.\d+$'
$token        = '(?<open><!--VERSION-->).*?(?<close><!--/VERSION-->)'
$bom          = [System.Text.UTF8Encoding]::new($true).GetPreamble()

if (-not (Test-Path -LiteralPath $versionFile)) {
    Write-Host "  No VERSION file; nothing stamped." -ForegroundColor Yellow
    exit 0
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch $versionShape) {
    Write-Host "  VERSION holds '$version', not MAJOR.MINOR.PATCH; nothing stamped." -ForegroundColor Red
    exit 1
}

$files = @(Get-ChildItem -LiteralPath $PSScriptRoot -Filter *.md -File)
if (Test-Path -LiteralPath $docsDir) {
    $files += @(Get-ChildItem -LiteralPath $docsDir -Recurse -File -Include *.html, *.md)
}

$stampedCount = 0
foreach ($file in $files) {
    # Keep a file's byte order mark exactly as found, present or absent, so a
    # stamp changes the version and nothing else.
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    $hasBom = $bytes.Length -ge $bom.Length -and
        -not (Compare-Object $bytes[0..($bom.Length - 1)] $bom -SyncWindow 0)
    $encoding = [System.Text.UTF8Encoding]::new($hasBom)

    $text = [System.IO.File]::ReadAllText($file.FullName, $encoding)
    $stamped = [regex]::Replace($text, $token, "`${open}$version`${close}")
    if ($stamped -ne $text) {
        [System.IO.File]::WriteAllText($file.FullName, $stamped, $encoding)
        Write-Host "  Stamped $version into $(Resolve-Path -Relative $file.FullName)"
        $stampedCount++
    }
}

if ($stampedCount -eq 0) {
    Write-Host "  Version stamps already read $version." -ForegroundColor Gray
}
# An explicit exit code: build.ps1 checks $LASTEXITCODE straight after this runs
# and would read a stale or empty one as a failure.
exit 0
