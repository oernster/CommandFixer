//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for the PATH lookup the engine is given, against a real directory and
// the real exec.LookPath. None of them call t.Parallel: they change the
// environment and the working directory of the whole process. That is also
// why t.Setenv refuses to run in a parallel test.

const (
	codeInstallCmd = "code --install-extension rust-lang.rust-analyzer"
	shimPathExt    = ".COM;.EXE;.BAT;.CMD"
	noCwdSearchVar = "NoDefaultCurrentDirectoryInExePath"
)

// shimDir makes a directory holding code.cmd, the shape VS Code installs.
func shimDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "code.cmd"), []byte("@echo off\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// enterDir makes dir the working directory until the test ends. It stands in
// for t.Chdir, which needs a newer Go than go.mod declares.
func enterDir(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

// The reported defect: suggest offered "mode ..." with code.cmd on PATH.
func TestNewEngine_CmdShimOnPath_NotCorrected(t *testing.T) {
	t.Setenv("PATH", shimDir(t))
	t.Setenv("PATHEXT", shimPathExt)
	cfg, _ := minimalConfig(t)
	if got, found := newEngine(cfg).Suggest(codeInstallCmd); found {
		t.Errorf("code.cmd is on PATH but was corrected to %q", got)
	}
}

// Take the shim off PATH and the same input is corrected, so the test above
// passes because of the lookup and not because the engine never offers mode.
func TestNewEngine_CmdShimNotOnPath_Corrected(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PATHEXT", shimPathExt)
	cfg, _ := minimalConfig(t)
	if _, found := newEngine(cfg).Suggest(codeInstallCmd); !found {
		t.Error("expected a correction when nothing named code is on PATH")
	}
}

// PowerShell will not run a program from the current directory without .\,
// so a code.cmd sitting there is not a command and does not count.
//
// exec.LookPath only looks in the current directory while
// NoDefaultCurrentDirectoryInExePath is absent. Some machines set it; with it
// present this test passed whatever onPath did. So it is removed here, with
// t.Setenv first so the original value comes back afterwards.
func TestOnPath_CurrentDirectoryOnly_DoesNotCount(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("PATHEXT", shimPathExt)
	t.Setenv(noCwdSearchVar, "")
	if err := os.Unsetenv(noCwdSearchVar); err != nil {
		t.Fatal(err)
	}
	enterDir(t, shimDir(t))
	if onPath("code") {
		t.Error("a shim found only in the current directory counted as on PATH")
	}
}
