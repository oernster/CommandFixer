package shell

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for the operations that change a user's PowerShell profile.
//
// Install, Uninstall and the snippet removal underneath them. These are
// the ones worth being careful about: a failure here does not stay in
// this directory, it runs on every prompt someone types.

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
