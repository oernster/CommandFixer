package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/commandfixer/shell"
)

// Tests for the two commands that modify a user's PowerShell profile.
//
// Their own file because they are the only part of this tool that changes
// something outside its own directory, and the only part whose failure
// leaves a dead hook line running on every prompt someone types.

// ---------------------------------------------------------------------------
// cmdInstall
// ---------------------------------------------------------------------------

func TestCmdInstall_WithExplicitProfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := cmdInstall([]string{profile}); err != nil {
		t.Fatalf("cmdInstall returned error: %v", err)
	}
	installed, err := shell.IsInstalled(profile)
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Error("expected profile to be installed")
	}
}

func TestCmdInstall_AlreadyInstalled_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := cmdInstall([]string{profile}); err != nil {
		t.Fatalf("first cmdInstall error: %v", err)
	}
	err := cmdInstall([]string{profile})
	if !errors.Is(err, shell.ErrAlreadyInstalled) {
		t.Errorf("expected ErrAlreadyInstalled, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// cmdUninstall
// ---------------------------------------------------------------------------

func TestCmdUninstall_WithExplicitProfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := cmdInstall([]string{profile}); err != nil {
		t.Fatalf("install setup error: %v", err)
	}
	if err := cmdUninstall([]string{profile}); err != nil {
		t.Fatalf("cmdUninstall returned error: %v", err)
	}
	installed, _ := shell.IsInstalled(profile)
	if installed {
		t.Error("expected profile to be uninstalled")
	}
}

func TestCmdUninstall_NotInstalled_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	if err := os.WriteFile(profile, []byte("# empty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := cmdUninstall([]string{profile})
	if !errors.Is(err, shell.ErrNotInstalled) {
		t.Errorf("expected ErrNotInstalled, got: %v", err)
	}
}
