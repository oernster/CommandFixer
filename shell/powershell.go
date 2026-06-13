// Package shell manages the PowerShell profile integration for CommandFixer.
package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	snippetStart = "# CommandFixer Integration - DO NOT EDIT"
	snippetEnd   = "# End CommandFixer Integration"
)

// ErrAlreadyInstalled is returned by Install when the hook is already present.
var ErrAlreadyInstalled = errors.New("CommandFixer already installed in PowerShell profile")

// ErrNotInstalled is returned by Uninstall when the hook is not present.
var ErrNotInstalled = errors.New("CommandFixer not found in PowerShell profile")

// ProfileSnippet returns the PowerShell block that intercepts the Enter key,
// checks the typed command against the fuzzy-matching engine, and prompts the
// user to confirm before applying any correction. The handler first verifies
// the binary still exists (Test-Path), so an uninstalled or moved executable
// fails silently instead of raising an error on every keystroke.
// binaryPath must be the full path to the commandfixer executable.
func ProfileSnippet(binaryPath string) string {
	return fmt.Sprintf(`%s
Set-PSReadLineKeyHandler -Key Enter -ScriptBlock {
    $cfBin = '%s'
    $line = $null; $cursor = $null
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    if ($line.Trim() -ne '' -and (Test-Path -LiteralPath $cfBin)) {
        $suggestion = & $cfBin suggest "$line" 2>$null
        if ($LASTEXITCODE -eq 0 -and $suggestion) {
            Write-Host ""
            Write-Host "CommandFixer: did you mean: $suggestion [Y/n] " -NoNewline -ForegroundColor Yellow
            $key = [Console]::ReadKey($true)
            Write-Host ""
            if ($key.KeyChar -eq 'y' -or $key.KeyChar -eq 'Y' -or $key.Key -eq 'Enter') {
                [Microsoft.PowerShell.PSConsoleReadLine]::RevertLine()
                [Microsoft.PowerShell.PSConsoleReadLine]::Insert($suggestion)
                & $cfBin log "$line" "$suggestion" 2>$null
            }
        }
    }
    [Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine()
}
%s
`, snippetStart, binaryPath, snippetEnd)
}

// DefaultProfilePath returns the CurrentUserAllHosts PowerShell 7 profile path.
// Typically: $HOME\Documents\PowerShell\profile.ps1
func DefaultProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, "Documents", "PowerShell", "profile.ps1"), nil
}

// PS5ProfilePath returns the CurrentUserAllHosts Windows PowerShell 5 profile path.
// Typically: $HOME\Documents\WindowsPowerShell\profile.ps1
func PS5ProfilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, "Documents", "WindowsPowerShell", "profile.ps1"), nil
}

// AllProfilePaths returns profile paths for all supported PowerShell versions:
// PowerShell 7 (pwsh) and Windows PowerShell 5 (powershell.exe).
// Install/uninstall into all returned paths to cover both shells.
func AllProfilePaths() ([]string, error) {
	ps7, err := DefaultProfilePath()
	if err != nil {
		return nil, err
	}
	ps5, err := PS5ProfilePath()
	if err != nil {
		return nil, err
	}
	return []string{ps7, ps5}, nil
}

// Install appends the CommandFixer hook snippet to the profile at profilePath.
// Creates parent directories if they do not exist.
// Returns ErrAlreadyInstalled if the snippet is already present.
func Install(profilePath, binaryPath string) error {
	existing, err := readProfileSafe(profilePath)
	if err != nil {
		return err
	}
	if strings.Contains(existing, snippetStart) {
		return ErrAlreadyInstalled
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}
	content := existing
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += ProfileSnippet(binaryPath)
	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write profile %q: %w", profilePath, err)
	}
	return nil
}

// Uninstall removes the CommandFixer hook snippet from the profile at profilePath.
// Returns ErrNotInstalled if no snippet is found.
func Uninstall(profilePath string) error {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("read profile %q: %w", profilePath, err)
	}
	content := string(data)
	if !strings.Contains(content, snippetStart) {
		return ErrNotInstalled
	}
	content = removeSnippet(content)
	if err := os.WriteFile(profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write profile %q: %w", profilePath, err)
	}
	return nil
}

// IsInstalled reports whether the CommandFixer snippet is present in profilePath.
// Returns (false, nil) if the profile file does not exist.
func IsInstalled(profilePath string) (bool, error) {
	content, err := readProfileSafe(profilePath)
	if err != nil {
		return false, err
	}
	return strings.Contains(content, snippetStart), nil
}

// readProfileSafe reads the profile file, returning "" when it does not exist.
func readProfileSafe(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read profile %q: %w", path, err)
	}
	return string(data), nil
}

// removeSnippet strips the CommandFixer block (start marker through end marker) from content.
func removeSnippet(content string) string {
	start := strings.Index(content, snippetStart)
	if start < 0 {
		return content
	}
	end := strings.Index(content[start:], snippetEnd)
	if end < 0 {
		// No closing marker found: trim from start marker onwards.
		return strings.TrimRight(content[:start], "\n") + "\n"
	}
	endPos := start + end + len(snippetEnd)
	// Consume one trailing newline after the end marker.
	if endPos < len(content) && content[endPos] == '\n' {
		endPos++
	}
	before := strings.TrimRight(content[:start], "\n")
	after := content[endPos:]
	switch {
	case before == "":
		return after
	case after == "":
		return before + "\n"
	default:
		return before + "\n" + after
	}
}
