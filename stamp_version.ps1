# Writes the version held in VERSION into every stamped static file.
#
# The GitHub Pages site cannot read VERSION when it is shown, so each place on
# it that names the version carries a token instead:
#
#     <!--VERSION-->MAJOR.MINOR.PATCH<!--/VERSION-->
#
# This rewrites whatever sits between the two markers in everything under
# docs/. The site is the only place the version is written out: the markdown
# documents at the root carry no version at all, so this deliberately never
# looks at them. It is idempotent: a file that already
# carries the current version is not written at all. build.ps1 runs it before
# every build, so a release cannot ship a site naming the previous version,
# which is exactly what happened when 1.4.0 was first committed.
#
# The site's local stylesheet and script links also carry their file's content
# hash, as styles.css?v=<hash>. GitHub Pages lets a browser keep a stylesheet
# for ten minutes, so a fresh page could otherwise be drawn with the old one;
# a changed file is a new address instead.
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

# A local stylesheet or script link: the path, any query it already carries
# (replaced), then any fragment (kept). The hash is the first
# $assetHashLength hex characters of the file's SHA-256.
$assetLink       = '(?<attr>\b(?:href|src)=)(?<quote>["''])(?<path>[^"''?#]+\.(?:css|js))(?:\?[^"''#]*)?(?<fragment>#[^"'']*)?\k<quote>'
$assetHashLength = 10

# The hash a link carries, over the file's bytes with CRLF read as LF, so a
# Windows checkout and the LF blob GitHub serves agree. A link to a missing
# file stops the stamp rather than hash nothing.
function Get-AssetHash([string]$path) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "A site page links $path, which does not exist; nothing hashed."
    }
    # Latin-1 maps every byte to one character, so the CRLF swap is exact.
    $latin1 = [System.Text.Encoding]::GetEncoding('iso-8859-1')
    $bytes = $latin1.GetBytes($latin1.GetString([System.IO.File]::ReadAllBytes($path)).Replace("`r`n", "`n"))
    $digest = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
    (-join ($digest | ForEach-Object { $_.ToString('x2') })).Substring(0, $assetHashLength)
}

# Puts each relative link's hash on it, resolved against the page's own folder.
# Remote, protocol-relative and root-absolute links are not this site's files
# and are left alone.
function Add-AssetHashes([string]$text, [string]$folder) {
    [regex]::Replace($text, $assetLink, {
        param($link)
        $path = $link.Groups['path'].Value
        if ($path.StartsWith('/') -or $path.Contains(':')) {
            return $link.Value
        }
        $quote = $link.Groups['quote'].Value
        '{0}{1}{2}?v={3}{4}{1}' -f $link.Groups['attr'].Value, $quote, $path,
            (Get-AssetHash (Join-Path $folder $path)), $link.Groups['fragment'].Value
    })
}

if (-not (Test-Path -LiteralPath $versionFile)) {
    Write-Host "  No VERSION file; nothing stamped." -ForegroundColor Yellow
    exit 0
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch $versionShape) {
    Write-Host "  VERSION holds '$version', not MAJOR.MINOR.PATCH; nothing stamped." -ForegroundColor Red
    exit 1
}

$files = @()
if (Test-Path -LiteralPath $docsDir) {
    $files = @(Get-ChildItem -LiteralPath $docsDir -Recurse -File -Include *.html, *.md)
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
    # Only the site's pages link stylesheets and scripts.
    $linked = $stamped
    if ($file.Extension -eq '.html') {
        $linked = Add-AssetHashes $stamped $file.DirectoryName
    }
    if ($linked -ne $text) {
        [System.IO.File]::WriteAllText($file.FullName, $linked, $encoding)
        if ($stamped -ne $text) {
            Write-Host "  Stamped $version into $(Resolve-Path -Relative $file.FullName)"
        }
        if ($linked -ne $stamped) {
            Write-Host "  Stamped asset hashes into $(Resolve-Path -Relative $file.FullName)"
        }
        $stampedCount++
    }
}

if ($stampedCount -eq 0) {
    Write-Host "  Version stamps already read $version; asset hashes are current." -ForegroundColor Gray
}
# An explicit exit code: build.ps1 checks $LASTEXITCODE straight after this runs
# and would read a stale or empty one as a failure.
exit 0
