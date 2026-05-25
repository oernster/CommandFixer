package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/commandfixer/config"
	"github.com/oernster/commandfixer/shell"
)

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

// minimalConfig returns a Config with one typo rule and a temp log path.
func minimalConfig(t *testing.T) (*config.Config, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "corrections.log")
	return &config.Config{
		Typos: []config.TypoEntry{
			{From: "git sattus", To: "git status"},
		},
		Settings: config.Settings{
			LogFile:     logPath,
			MaxLogLines: 100,
		},
	}, logPath
}

// ---------------------------------------------------------------------------
// run (smoke test - exercises DefaultConfigPath resolution)
// ---------------------------------------------------------------------------

func TestRun_HelpFlag(t *testing.T) {
	t.Parallel()
	// run() resolves the real default config path, then calls dispatch.
	// We only verify it does not return an error for the help command.
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

// ---------------------------------------------------------------------------
// cmdCorrect
// ---------------------------------------------------------------------------

func TestCmdCorrect_NoArgs(t *testing.T) {
	t.Parallel()
	err := cmdCorrect([]string{}, "")
	if err == nil {
		t.Fatal("expected error when no command provided")
	}
}

func TestCmdCorrect_NoMatch(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// "git status" already correct - no correction, no log write.
	if err := cmdCorrect([]string{"git", "status"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect returned error: %v", err)
	}
}

func TestCmdCorrect_Match_WritesLog(t *testing.T) {
	t.Parallel()
	cfg, logPath := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)

	if err := cmdCorrect([]string{"git", "sattus"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect returned error: %v", err)
	}
	// Log file must have been created.
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file not created after correction: %v", err)
	}
}

func TestCmdCorrect_MultiWordInput(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// Extra args get joined back into the command.
	if err := cmdCorrect([]string{"git", "sattus", "-v"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect returned error: %v", err)
	}
}

func TestCmdCorrect_MissingConfigFile_UsesDefault(t *testing.T) {
	t.Parallel()
	// Point to a non-existent config file: LoadOrDefault returns empty config.
	cfgPath := filepath.Join(t.TempDir(), "nonexistent.json")
	if err := cmdCorrect([]string{"git", "status"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect with missing config returned error: %v", err)
	}
}

func TestCmdCorrect_BadConfig_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte("NOT JSON"), 0644); err != nil {
		t.Fatal(err)
	}
	err := cmdCorrect([]string{"git", "sattus"}, cfgPath)
	if err == nil {
		t.Fatal("expected error for bad config, got nil")
	}
}

func TestCmdCorrect_InvalidRegex_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Typos: []config.TypoEntry{
			{From: `[bad(`, To: "x", Regex: true},
		},
		Settings: config.Settings{
			LogFile:     filepath.Join(t.TempDir(), "log.log"),
			MaxLogLines: 100,
		},
	}
	cfgPath := writeTempConfig(t, cfg)
	err := cmdCorrect([]string{"something"}, cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

// ---------------------------------------------------------------------------
// cmdInstall
// ---------------------------------------------------------------------------

func TestCmdInstall_WithExplicitProfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	profile := filepath.Join(dir, "profile.ps1")
	// Pass profile path as arg[0] to override the default.
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

// ---------------------------------------------------------------------------
// cmdStats
// ---------------------------------------------------------------------------

func TestCmdStats_EmptyLog(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// Log file does not exist yet: should show 0 corrections without error.
	if err := cmdStats(cfgPath); err != nil {
		t.Fatalf("cmdStats returned error: %v", err)
	}
}

func TestCmdStats_WithEntries(t *testing.T) {
	t.Parallel()
	cfg, logPath := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// Populate the log via cmdCorrect.
	if err := cmdCorrect([]string{"git", "sattus"}, cfgPath); err != nil {
		t.Fatal(err)
	}
	_ = logPath
	if err := cmdStats(cfgPath); err != nil {
		t.Fatalf("cmdStats with entries returned error: %v", err)
	}
}

func TestCmdStats_BadConfig_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(cfgPath, []byte("NOTJSON"), 0644); err != nil {
		t.Fatal(err)
	}
	err := cmdStats(cfgPath)
	if err == nil {
		t.Fatal("expected error for bad config, got nil")
	}
}

// TestCmdStats_ReadStatsError exercises the logger.ReadStats error path in cmdStats.
// We set log_file to a directory path, which causes os.ReadFile to fail with
// a non-IsNotExist error.
func TestCmdStats_ReadStatsError(t *testing.T) {
	t.Parallel()
	// logDir is a directory, not a file: ReadStats will fail on it.
	logDir := t.TempDir()
	cfg := &config.Config{
		Typos: nil,
		Settings: config.Settings{
			LogFile:     logDir,
			MaxLogLines: 100,
		},
	}
	cfgPath := writeTempConfig(t, cfg)
	err := cmdStats(cfgPath)
	if err == nil {
		t.Fatal("expected error when log path is a directory, got nil")
	}
	if !strings.Contains(err.Error(), "read stats") {
		t.Errorf("expected 'read stats' in error message, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// printUsage (smoke)
// ---------------------------------------------------------------------------

func TestPrintUsage(t *testing.T) {
	t.Parallel()
	// Just verify it does not panic.
	printUsage()
}
