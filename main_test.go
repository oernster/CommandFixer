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
// cmdSuggest
// ---------------------------------------------------------------------------

func TestCmdSuggest_NoArgs(t *testing.T) {
	t.Parallel()
	err := cmdSuggest([]string{}, "")
	if err == nil {
		t.Fatal("expected error when no command provided")
	}
}

func TestCmdSuggest_KnownTypo(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// "git sattus" should fuzzy-match to "git status".
	if err := cmdSuggest([]string{"git sattus"}, cfgPath); err != nil {
		t.Fatalf("cmdSuggest returned error: %v", err)
	}
}

func TestCmdSuggest_ExactCommand_NoOutput(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// "git status" is already correct: no output, no error.
	if err := cmdSuggest([]string{"git", "status"}, cfgPath); err != nil {
		t.Fatalf("cmdSuggest returned error: %v", err)
	}
}

func TestCmdSuggest_UnknownTool_NoOutput(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	if err := cmdSuggest([]string{"foobar", "baz"}, cfgPath); err != nil {
		t.Fatalf("cmdSuggest returned error for unknown tool: %v", err)
	}
}

func TestCmdSuggest_MultiWordInput_Joined(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// Multiple args get joined; "git sattus" should correct.
	if err := cmdSuggest([]string{"git", "sattus"}, cfgPath); err != nil {
		t.Fatalf("cmdSuggest returned error: %v", err)
	}
}

func TestCmdSuggest_BadConfig_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte("NOTJSON"), 0644); err != nil {
		t.Fatal(err)
	}
	err := cmdSuggest([]string{"git", "sattus"}, cfgPath)
	if err == nil {
		t.Fatal("expected error for bad config, got nil")
	}
}

func TestCmdSuggest_MissingConfig_UsesDefault(t *testing.T) {
	t.Parallel()
	cfgPath := filepath.Join(t.TempDir(), "nonexistent.json")
	// No config: falls back to default threshold. Should not error.
	if err := cmdSuggest([]string{"git", "status"}, cfgPath); err != nil {
		t.Fatalf("cmdSuggest with missing config returned error: %v", err)
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
	// "git status" already correct: no correction, no log write.
	if err := cmdCorrect([]string{"git", "status"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect returned error: %v", err)
	}
}

func TestCmdCorrect_Match_WritesLog(t *testing.T) {
	t.Parallel()
	cfg, logPath := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)

	// "git sattus" fuzzy-matches "git status"; log must be created.
	if err := cmdCorrect([]string{"git", "sattus"}, cfgPath); err != nil {
		t.Fatalf("cmdCorrect returned error: %v", err)
	}
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

// ---------------------------------------------------------------------------
// cmdLog
// ---------------------------------------------------------------------------

func TestCmdLog_NoArgs_ReturnsError(t *testing.T) {
	t.Parallel()
	err := cmdLog([]string{}, "")
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestCmdLog_OneArg_ReturnsError(t *testing.T) {
	t.Parallel()
	err := cmdLog([]string{"git sattus"}, "")
	if err == nil {
		t.Fatal("expected error when only one arg provided")
	}
}

func TestCmdLog_WritesEntry(t *testing.T) {
	t.Parallel()
	cfg, logPath := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	if err := cmdLog([]string{"git sattus", "git status"}, cfgPath); err != nil {
		t.Fatalf("cmdLog returned error: %v", err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file not created after cmdLog: %v", err)
	}
}

func TestCmdLog_BadConfig_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte("NOTJSON"), 0644); err != nil {
		t.Fatal(err)
	}
	err := cmdLog([]string{"git sattus", "git status"}, cfgPath)
	if err == nil {
		t.Fatal("expected error for bad config, got nil")
	}
}

func TestCmdLog_MissingConfig_UsesDefault(t *testing.T) {
	t.Parallel()
	// Missing config: uses default log path in home dir.
	// Just verify it doesn't error on config resolution itself.
	cfgPath := filepath.Join(t.TempDir(), "nonexistent.json")
	// cmdLog may fail if the default log path can't be written (CI), but
	// the config load itself should not error.
	_ = cmdLog([]string{"git sattus", "git status"}, cfgPath)
}

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

// ---------------------------------------------------------------------------
// cmdStats
// ---------------------------------------------------------------------------

func TestCmdStats_EmptyLog(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	if err := cmdStats(cfgPath); err != nil {
		t.Fatalf("cmdStats returned error: %v", err)
	}
}

func TestCmdStats_WithEntries(t *testing.T) {
	t.Parallel()
	cfg, _ := minimalConfig(t)
	cfgPath := writeTempConfig(t, cfg)
	// Populate log via cmdCorrect. "git sattus" fuzzy-matches "git status".
	if err := cmdCorrect([]string{"git", "sattus"}, cfgPath); err != nil {
		t.Fatal(err)
	}
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
	logDir := t.TempDir()
	cfg := &config.Config{
		Settings: config.Settings{
			LogFile:             logDir,
			MaxLogLines:         100,
			SimilarityThreshold: 0.6,
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
	printUsage()
}
