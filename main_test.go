package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/commandfixer/config"
)

// Tests for the CLI surface: argument routing and the shared helpers.
//
// The per-command behaviour lives in commands_test.go, and the two
// commands that write to a real user's PowerShell profile in
// install_test.go, where they are easier to find and harder to overlook.

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeTempConfig writes a config.json to a temp dir and returns the file path.
func writeTempConfig(t *testing.T, cfg *config.Config) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal test config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}

// minimalConfig returns a Config with default threshold and a temp log path.
func minimalConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "corrections.log")
	return &config.Config{
		Settings: config.Settings{
			LogFile:             logPath,
			MaxLogLines:         100,
			SimilarityThreshold: 0.6,
		},
	}, logPath
}

// ---------------------------------------------------------------------------
// run (smoke test - exercises DefaultConfigPath resolution)
// ---------------------------------------------------------------------------

func TestRun_HelpFlag(t *testing.T) {
	t.Parallel()
	if err := run([]string{"help"}); err != nil {
		t.Fatalf("run help returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// dispatch - routing
// ---------------------------------------------------------------------------

func TestDispatch_NoArgs_PrintsUsage(t *testing.T) {
	t.Parallel()
	cfgPath := writeTempConfig(t, &config.Config{})
	if err := dispatch([]string{}, cfgPath); err != nil {
		t.Fatalf("dispatch with no args returned error: %v", err)
	}
}

func TestDispatch_Help(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"help"}, ""); err != nil {
		t.Fatalf("dispatch help returned error: %v", err)
	}
}

func TestDispatch_HelpAlias_DoubleDash(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"--help"}, ""); err != nil {
		t.Fatalf("dispatch --help returned error: %v", err)
	}
}

func TestDispatch_HelpAlias_DashH(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"-h"}, ""); err != nil {
		t.Fatalf("dispatch -h returned error: %v", err)
	}
}

func TestDispatch_Version(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"version"}, ""); err != nil {
		t.Fatalf("dispatch version returned error: %v", err)
	}
}

func TestDispatch_VersionAlias_DoubleDash(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"--version"}, ""); err != nil {
		t.Fatalf("dispatch --version returned error: %v", err)
	}
}

func TestDispatch_VersionAlias_DashV(t *testing.T) {
	t.Parallel()
	if err := dispatch([]string{"-v"}, ""); err != nil {
		t.Fatalf("dispatch -v returned error: %v", err)
	}
}

func TestDispatch_UnknownCommand(t *testing.T) {
	t.Parallel()
	err := dispatch([]string{"frobnicate"}, "")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("error does not mention 'unknown command': %v", err)
	}
}

func TestDispatch_Suggest(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	if err := dispatch([]string{"suggest", "git", "status"}, cfgPath); err != nil {
		t.Fatalf("dispatch suggest returned error: %v", err)
	}
}

func TestDispatch_Log(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	if err := dispatch([]string{"log", "git sattus", "git status"}, cfgPath); err != nil {
		t.Fatalf("dispatch log returned error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printUsage (smoke)
// ---------------------------------------------------------------------------

func TestPrintUsage(t *testing.T) {
	t.Parallel()
	printUsage()
}
