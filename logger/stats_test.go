package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	t.Parallel()
	l := New("/tmp/test.log")
	if l == nil {
		t.Fatal("expected non-nil Logger")
	}
	if l.logPath != "/tmp/test.log" {
		t.Errorf("expected logPath %q, got %q", "/tmp/test.log", l.logPath)
	}
}

// ---------------------------------------------------------------------------
// Log
// ---------------------------------------------------------------------------

func TestLog_WritesEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")
	l := New(logPath)

	err := l.Log("git sattus", "git status", "git sattus -> git status")
	if err != nil {
		t.Fatalf("Log returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, "git sattus") {
		t.Errorf("log entry missing original: %q", line)
	}
	if !strings.Contains(line, "git status") {
		t.Errorf("log entry missing corrected: %q", line)
	}
}

func TestLog_CreatesParentDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Sub-directory does not exist yet.
	logPath := filepath.Join(dir, "sub", "deep", "corrections.log")
	l := New(logPath)
	if err := l.Log("a", "b", "a -> b"); err != nil {
		t.Fatalf("Log returned error: %v", err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("log file not created: %v", err)
	}
}

func TestLog_AppendsMultipleEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")
	l := New(logPath)

	for i := 0; i < 3; i++ {
		if err := l.Log("orig", "corrected", "rule"); err != nil {
			t.Fatalf("Log[%d] error: %v", i, err)
		}
	}
	data, _ := os.ReadFile(logPath)
	lines := splitLines(string(data))
	if len(lines) != 3 {
		t.Errorf("expected 3 log lines, got %d", len(lines))
	}
}

func TestLog_EntriesContainTimestamp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")
	l := New(logPath)
	before := time.Now().UTC()
	if err := l.Log("a", "b", "rule"); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC()

	stats, err := ReadStats(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.History) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(stats.History))
	}
	ts := stats.History[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", ts, before, after)
	}
}

// ---------------------------------------------------------------------------
// ReadStats
// ---------------------------------------------------------------------------

func TestReadStats_FileNotExist(t *testing.T) {
	t.Parallel()
	stats, err := ReadStats(filepath.Join(t.TempDir(), "no.log"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if stats.TotalCorrections != 0 {
		t.Errorf("expected 0 corrections, got %d", stats.TotalCorrections)
	}
	if stats.RuleCounts == nil {
		t.Error("RuleCounts must not be nil")
	}
}

func TestReadStats_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")
	if err := os.WriteFile(logPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	stats, err := ReadStats(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalCorrections != 0 {
		t.Errorf("expected 0 corrections, got %d", stats.TotalCorrections)
	}
}

func TestReadStats_ValidEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")
	l := New(logPath)

	if err := l.Log("git sattus", "git status", "git sattus -> git status"); err != nil {
		t.Fatal(err)
	}
	if err := l.Log("git sattus", "git status", "git sattus -> git status"); err != nil {
		t.Fatal(err)
	}
	if err := l.Log("ls --afl", "ls -afl", "ls --afl -> ls -afl"); err != nil {
		t.Fatal(err)
	}

	stats, err := ReadStats(logPath)
	if err != nil {
		t.Fatalf("ReadStats error: %v", err)
	}
	if stats.TotalCorrections != 3 {
		t.Errorf("expected 3 total corrections, got %d", stats.TotalCorrections)
	}
	if stats.RuleCounts["git sattus -> git status"] != 2 {
		t.Errorf("expected rule count 2, got %d", stats.RuleCounts["git sattus -> git status"])
	}
	if len(stats.History) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(stats.History))
	}
}

// TestReadStats_ReadError exercises the non-IsNotExist error branch.
// Passing a directory as the log path causes os.ReadFile to fail with an
// error that is NOT os.IsNotExist.
func TestReadStats_ReadError(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	_, err := ReadStats(dirPath)
	if err == nil {
		t.Fatal("expected error when log path is a directory, got nil")
	}
}

func TestReadStats_MalformedLinesSkipped(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")

	// Write one valid entry and one malformed line.
	validJSON := `{"timestamp":"2024-01-01T00:00:00Z","original":"a","corrected":"b","rule":"r"}`
	content := validJSON + "\nNOT_VALID_JSON\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := ReadStats(logPath)
	if err != nil {
		t.Fatalf("ReadStats returned error: %v", err)
	}
	// Only the valid line counts.
	if stats.TotalCorrections != 1 {
		t.Errorf("expected 1 correction (malformed line skipped), got %d", stats.TotalCorrections)
	}
}

func TestReadStats_RuleCountsAccurate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "corrections.log")
	l := New(logPath)

	rules := []string{"ruleA", "ruleA", "ruleB"}
	for _, r := range rules {
		if err := l.Log("x", "y", r); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := ReadStats(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if stats.RuleCounts["ruleA"] != 2 {
		t.Errorf("expected ruleA count 2, got %d", stats.RuleCounts["ruleA"])
	}
	if stats.RuleCounts["ruleB"] != 1 {
		t.Errorf("expected ruleB count 1, got %d", stats.RuleCounts["ruleB"])
	}
}

// ---------------------------------------------------------------------------
// splitLines
// ---------------------------------------------------------------------------

func TestSplitLines_Empty(t *testing.T) {
	t.Parallel()
	if lines := splitLines(""); len(lines) != 0 {
		t.Errorf("expected empty slice, got %v", lines)
	}
}

func TestSplitLines_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	if lines := splitLines("  \n  \n  "); len(lines) != 0 {
		t.Errorf("expected empty slice for whitespace-only, got %v", lines)
	}
}

func TestSplitLines_NormalLines(t *testing.T) {
	t.Parallel()
	lines := splitLines("line1\nline2\nline3")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	t.Parallel()
	lines := splitLines("line1\nline2\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}
