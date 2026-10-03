# The knowledge install.ps1 and uninstall.ps1 both need, defined once.
#
# Dot-source this rather than restating any of it:
#
#     . (Join-Path $PSScriptRoot 'profile-hook.ps1')
#
# The marker strings and the profile paths also exist in Go (in
# shell/powershell.go) and they have to: the binary writes the hook; this
# file is what removes it when the binary is already gone. Two languages cannot
# share one literal, so shell/markers_test.go reads THIS file and fails if the
# two ever disagree. That is the point of the exercise. A marker changed on one
# side only would leave a hook line in someone's profile that nothing can find
# to remove, running on every prompt they type.

# Where the tool installs itself.
$CommandFixerBinaryName = 'commandfixer.exe'
$CommandFixerInstallDir = "$env:LOCALAPPDATA\CommandFixer"
$CommandFixerConfigDir  = "$env:USERPROFILE\.typo-fixer"

# The fences around the hook block in a user's profile. Everything between them
# belongs to CommandFixer and nothing outside them does.
$CommandFixerSnippetStart = '# CommandFixer Integration - DO NOT EDIT'
$CommandFixerSnippetEnd   = '# End CommandFixer Integration'

function Get-CommandFixerDocumentsFolder {
    <#
    .SYNOPSIS
        The folder PowerShell keeps its profiles under.
    .DESCRIPTION
        Asked of Windows rather than built from $HOME, because a Documents
        folder moved by OneDrive folder backup or redirection moves the
        profiles with it. $HOME\Documents is only the fallback when Windows
        gives no answer, the same rule as documentsDir in Go.
    #>
    $documents = [Environment]::GetFolderPath('MyDocuments')
    if (-not $documents) {
        $documents = Join-Path $HOME 'Documents'
    }
    $documents
}

function Get-CommandFixerProfilePaths {
    <#
    .SYNOPSIS
        The CurrentUserAllHosts profiles the hook is installed into.
    .DESCRIPTION
        PowerShell 7 (pwsh) first, then Windows PowerShell 5 (powershell.exe),
        the same order and the same locations as shell.AllProfilePaths in Go.
        Both are covered because the hook is installed into both.
    #>
    $documents = Get-CommandFixerDocumentsFolder
    @(
        (Join-Path $documents 'PowerShell\profile.ps1'),
        (Join-Path $documents 'WindowsPowerShell\profile.ps1')
    )
}
