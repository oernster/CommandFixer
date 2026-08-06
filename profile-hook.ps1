# The knowledge install.ps1 and uninstall.ps1 both need, defined once.
#
# Dot-source this rather than restating any of it:
#
#     . (Join-Path $PSScriptRoot 'profile-hook.ps1')
#
# The marker strings and the profile paths also exist in Go, in
# shell/powershell.go, and they have to: the binary writes the hook, and this
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

function Get-CommandFixerProfilePaths {
    <#
    .SYNOPSIS
        The CurrentUserAllHosts profiles the hook is installed into.
    .DESCRIPTION
        PowerShell 7 (pwsh) first, then Windows PowerShell 5 (powershell.exe),
        the same order and the same locations as shell.AllProfilePaths in Go.
        Both are covered because the hook is installed into both.
    #>
    @(
        (Join-Path $HOME 'Documents\PowerShell\profile.ps1'),
        (Join-Path $HOME 'Documents\WindowsPowerShell\profile.ps1')
    )
}
