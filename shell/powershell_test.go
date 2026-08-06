package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for what the profile integration is, and for reading it back.
//
// The snippet itself, the profile paths, and the two read-only
// operations. Nothing here writes to a profile: that is install_test.go.

// ---------------------------------------------------------------------------
// ProfileSnippet
// ---------------------------------------------------------------------------

func TestProfileSnippet_ContainsMarkers(t *testing.T) {
	t.Parallel()
	snip := ProfileSnippet(`C:\tools\commandfixer.exe`)
	if !strings.Contains(snip, snippetStart) {
		t.Errorf("snippet missing start marker %q", snippetStart)
	}
	if !strings.Contains(snip, snippetEnd) {
		t.Errorf("snippet missing end marker %q", snippetEnd)
	}
}

func TestProfileSnippet_ContainsBinaryPath(t *testing.T) {
	t.Parallel()
	bin := `C:\tools\commandfixer.exe`
	snip := ProfileSnippet(bin)
	if !strings.Contains(snip, bin) {
		t.Errorf("snippet does not contain binary path %q", bin)
	}
}

func TestProfileSnippet_ContainsPSReadLineCall(t *testing.T) {
	t.Parallel()
	snip := ProfileSnippet(`bin`)
	if !strings.Contains(snip, "Set-PSReadLineKeyHandler") {
		t.Error("snippet missing Set-PSReadLineKeyHandler call")
	}
}

func TestProfileSnippet_ContainsCompletenessGuard(t *testing.T) {
	t.Parallel()
	// The hook must parse the final buffer and refuse to submit incomplete
	// input, so a stray trailing backtick never opens the '>>' continuation
	// prompt. The guard is identified by the parser call and the exact
	// IncompleteInput signal the console host uses.
	snip := ProfileSnippet(`C:\tools\commandfixer.exe`)
	if !strings.Contains(snip, "ParseInput") {
		t.Error("snippet missing Parser.ParseInput completeness check")
	}
	if !strings.Contains(snip, "IncompleteInput") {
		t.Error("snippet missing IncompleteInput guard")
	}
	// The backtick must be expressed as [char]96, never a literal backtick
	// (the snippet is a Go raw string and could not contain one anyway).
	if !strings.Contains(snip, "[char]96") {
		t.Error("snippet missing [char]96 backtick strip in the repair pass")
	}
}

func TestProfileSnippet_GuardsMissingBinary(t *testing.T) {
	t.Parallel()
	// The hook must verify the binary exists before invoking it, so an
	// uninstalled or moved executable cannot error on every keystroke.
	snip := ProfileSnippet(`C:\tools\commandfixer.exe`)
	if !strings.Contains(snip, "Test-Path") {
		t.Error("snippet missing Test-Path guard for the binary path")
	}
}

// ---------------------------------------------------------------------------
// DefaultProfilePath
// ---------------------------------------------------------------------------

func TestDefaultProfilePath(t *testing.T) {
	t.Parallel()
	path, err := DefaultProfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "profile.ps1") {
		t.Errorf("expected suffix profile.ps1, got %q", path)
	}
	if !strings.Contains(path, "PowerShell") {
		t.Errorf("expected PowerShell in path, got %q", path)
	}
}

func TestPS5ProfilePath(t *testing.T) {
	t.Parallel()
	path, err := PS5ProfilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "profile.ps1") {
		t.Errorf("expected suffix profile.ps1, got %q", path)
	}
	if !strings.Contains(path, "WindowsPowerShell") {
		t.Errorf("expected WindowsPowerShell in path, got %q", path)
	}
}

func TestAllProfilePaths_ReturnsBoth(t *testing.T) {
	t.Parallel()
	paths, err := AllProfilePaths()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 profile paths, got %d", len(paths))
	}
	// First path: PS7 (Documents\PowerShell\profile.ps1)
	if !strings.Contains(paths[0], "PowerShell") || strings.Contains(paths[0], "Windows") {
		t.Errorf("expected PS7 path first (no 'Windows' prefix), got %q", paths[0])
	}
	// Second path: PS5 (Documents\WindowsPowerShell\profile.ps1)
	if !strings.Contains(paths[1], "WindowsPowerShell") {
		t.Errorf("expected PS5 path second (WindowsPowerShell), got %q", paths[1])
	}
}

// ---------------------------------------------------------------------------
// IsInstalled
// ---------------------------------------------------------------------------

func TestIsInstalled_True(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := Install(profile, "cfx.exe"); err != nil {
		t.Fatal(err)
	}
	installed, err := IsInstalled(profile)
	if err != nil {
		t.Fatalf("IsInstalled error: %v", err)
	}
	if !installed {
		t.Error("expected IsInstalled=true after Install")
	}
}

func TestIsInstalled_False_NoSnippet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := os.WriteFile(profile, []byte("# no snippet\n"), 0644); err != nil {
		t.Fatal(err)
	}
	installed, err := IsInstalled(profile)
	if err != nil {
		t.Fatalf("IsInstalled error: %v", err)
	}
	if installed {
		t.Error("expected IsInstalled=false when snippet absent")
	}
}

func TestIsInstalled_False_ProfileMissing(t *testing.T) {
	t.Parallel()
	installed, err := IsInstalled(filepath.Join(t.TempDir(), "missing.ps1"))
	if err != nil {
		t.Fatalf("expected nil error for missing profile, got: %v", err)
	}
	if installed {
		t.Error("expected IsInstalled=false for missing profile")
	}
}

// TestIsInstalled_ReadError exercises the IsInstalled error path.
// A directory path causes readProfileSafe to fail with a non-IsNotExist error.
func TestIsInstalled_ReadError(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	_, err := IsInstalled(dirPath)
	if err == nil {
		t.Fatal("expected error when profile path is a directory, got nil")
	}
}

// ---------------------------------------------------------------------------
// readProfileSafe
// ---------------------------------------------------------------------------

func TestReadProfileSafe_FileNotExist(t *testing.T) {
	t.Parallel()
	content, err := readProfileSafe(filepath.Join(t.TempDir(), "nope.ps1"))
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty string, got %q", content)
	}
}

func TestReadProfileSafe_ExistingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.ps1")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	content, err := readProfileSafe(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "hello" {
		t.Errorf("expected %q, got %q", "hello", content)
	}
}

// TestReadProfileSafe_OtherReadError exercises the non-IsNotExist error branch.
// Passing a directory path causes os.ReadFile to fail with an error that is
// NOT os.IsNotExist.
func TestReadProfileSafe_OtherReadError(t *testing.T) {
	t.Parallel()
	// A directory path causes os.ReadFile to return a non-not-found error.
	dirPath := t.TempDir()
	_, err := readProfileSafe(dirPath)
	if err == nil {
		t.Fatal("expected error when path is a directory, got nil")
	}
	if os.IsNotExist(err) {
		t.Errorf("error should NOT be IsNotExist; got: %v", err)
	}
}

// TestInstall_ReadProfileError exercises the Install error path when
// readProfileSafe returns a non-not-found error (directory as profile path).
func TestInstall_ReadProfileError(t *testing.T) {
	t.Parallel()
	// Pass an existing directory as the profile path.
	dirPath := t.TempDir()
	err := Install(dirPath, "cfx.exe")
	if err == nil {
		t.Fatal("expected error when profile path is a directory, got nil")
	}
}
