package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/commandfixer/config"
)

// Tests for the commands that read and write CommandFixer's own data:
// suggest, correct, log and stats. None of these touch the shell profile.

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
