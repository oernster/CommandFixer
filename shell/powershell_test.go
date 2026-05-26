package shell

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
// Install
// ---------------------------------------------------------------------------

func TestInstall_FreshProfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	bin := `C:\tools\commandfixer.exe`

	if err := Install(profile, bin); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("ReadFile after Install: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, snippetStart) {
		t.Error("profile missing start marker after Install")
	}
	if !strings.Contains(content, bin) {
		t.Error("profile missing binary path after Install")
	}
}

func TestInstall_ExistingProfileGetsAppended(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	existing := "# My existing profile\n$env:FOO = 'bar'\n"
	if err := os.WriteFile(profile, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Install(profile, "cfx.exe"); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	data, _ := os.ReadFile(profile)
	content := string(data)
	if !strings.Contains(content, "My existing profile") {
		t.Error("existing profile content was removed")
	}
	if !strings.Contains(content, snippetStart) {
		t.Error("snippet not appended to existing profile")
	}
}

func TestInstall_ExistingProfileNoTrailingNewline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	// No trailing newline.
	if err := os.WriteFile(profile, []byte("Write-Host 'hello'"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Install(profile, "cfx.exe"); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	data, _ := os.ReadFile(profile)
	// Should still have the original line followed by the snippet.
	content := string(data)
	if !strings.Contains(content, "Write-Host") {
		t.Error("existing line missing after install")
	}
	if !strings.Contains(content, snippetStart) {
		t.Error("snippet missing after install on no-newline profile")
	}
}

func TestInstall_AlreadyInstalled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	bin := "cfx.exe"

	// First install.
	if err := Install(profile, bin); err != nil {
		t.Fatalf("first Install error: %v", err)
	}
	// Second install should return ErrAlreadyInstalled.
	err := Install(profile, bin)
	if !errors.Is(err, ErrAlreadyInstalled) {
		t.Errorf("expected ErrAlreadyInstalled, got: %v", err)
	}
}

func TestInstall_CreatesParentDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Profile lives inside a subdirectory that does not exist yet.
	profile := filepath.Join(dir, "sub", "deep", "profile.ps1")
	if err := Install(profile, "cfx.exe"); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if _, err := os.Stat(profile); err != nil {
		t.Errorf("profile file not created: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Uninstall
// ---------------------------------------------------------------------------

func TestUninstall_RemovesSnippet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	bin := "cfx.exe"

	if err := Install(profile, bin); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(profile); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}
	data, _ := os.ReadFile(profile)
	if strings.Contains(string(data), snippetStart) {
		t.Error("snippet still present after Uninstall")
	}
}

func TestUninstall_PreservesExistingContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	existing := "# My profile\n"
	if err := os.WriteFile(profile, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Install(profile, "cfx.exe"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(profile); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}
	data, _ := os.ReadFile(profile)
	if !strings.Contains(string(data), "My profile") {
		t.Error("existing content removed by Uninstall")
	}
}

func TestUninstall_NotInstalled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := os.WriteFile(profile, []byte("# something else\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := Uninstall(profile)
	if !errors.Is(err, ErrNotInstalled) {
		t.Errorf("expected ErrNotInstalled, got: %v", err)
	}
}

func TestUninstall_FileNotFound(t *testing.T) {
	t.Parallel()
	err := Uninstall(filepath.Join(t.TempDir(), "no_profile.ps1"))
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
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
// removeSnippet (internal, tested via exported Install/Uninstall)
// We test edge cases of removeSnippet directly since it has branching logic.
// ---------------------------------------------------------------------------

func TestRemoveSnippet_NoStartMarker(t *testing.T) {
	t.Parallel()
	content := "# just some content\n"
	result := removeSnippet(content)
	if result != content {
		t.Errorf("expected unchanged content, got %q", result)
	}
}

func TestRemoveSnippet_NoEndMarker(t *testing.T) {
	t.Parallel()
	// Start marker present but no end marker.
	content := "# before\n" + snippetStart + "\nsome stuff without end"
	result := removeSnippet(content)
	if strings.Contains(result, snippetStart) {
		t.Error("start marker still present after removeSnippet with missing end")
	}
	if !strings.Contains(result, "# before") {
		t.Error("content before snippet was removed unexpectedly")
	}
}

func TestRemoveSnippet_SnippetAtStart(t *testing.T) {
	t.Parallel()
	content := snippetStart + "\nstuff\n" + snippetEnd + "\n# after\n"
	result := removeSnippet(content)
	if strings.Contains(result, snippetStart) {
		t.Error("start marker present after removal")
	}
	if !strings.Contains(result, "# after") {
		t.Error("content after snippet was removed")
	}
}

func TestRemoveSnippet_SnippetAtEnd(t *testing.T) {
	t.Parallel()
	content := "# before\n" + snippetStart + "\nstuff\n" + snippetEnd + "\n"
	result := removeSnippet(content)
	if strings.Contains(result, snippetStart) {
		t.Error("start marker present after removal")
	}
	if !strings.Contains(result, "# before") {
		t.Error("content before snippet removed")
	}
}

func TestRemoveSnippet_EmptyBeforeAndAfter(t *testing.T) {
	t.Parallel()
	content := snippetStart + "\nstuff\n" + snippetEnd + "\n"
	result := removeSnippet(content)
	if strings.Contains(result, snippetStart) {
		t.Error("marker still present")
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
