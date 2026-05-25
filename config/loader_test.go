package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultConfigDir / DefaultConfigPath
// ---------------------------------------------------------------------------

func TestDefaultConfigDir(t *testing.T) {
	t.Parallel()
	dir, err := DefaultConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(dir, ".typo-fixer") {
		t.Errorf("expected suffix .typo-fixer, got %q", dir)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	t.Parallel()
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("expected suffix config.json, got %q", path)
	}
	if !strings.Contains(path, ".typo-fixer") {
		t.Errorf("expected .typo-fixer in path, got %q", path)
	}
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestLoad_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	content := `{
		"settings": {
			"log_file": "/tmp/test.log",
			"max_log_lines": 500,
			"similarity_threshold": 0.8
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Settings.LogFile != "/tmp/test.log" {
		t.Errorf("unexpected LogFile: %q", cfg.Settings.LogFile)
	}
	if cfg.Settings.MaxLogLines != 500 {
		t.Errorf("expected MaxLogLines 500, got %d", cfg.Settings.MaxLogLines)
	}
	if cfg.Settings.SimilarityThreshold != 0.8 {
		t.Errorf("expected SimilarityThreshold 0.8, got %v", cfg.Settings.SimilarityThreshold)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Load wraps the error with fmt.Errorf(%w); use errors.Is to unwrap correctly.
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist (unwrapped), got: %v", err)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestLoad_AppliesDefaults_WhenFieldsMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Empty settings: all defaults should be applied.
	if err := os.WriteFile(path, []byte(`{"settings":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Settings.LogFile == "" {
		t.Error("expected default LogFile to be set")
	}
	if cfg.Settings.MaxLogLines != 10000 {
		t.Errorf("expected default MaxLogLines 10000, got %d", cfg.Settings.MaxLogLines)
	}
	if cfg.Settings.SimilarityThreshold != 0.6 {
		t.Errorf("expected default SimilarityThreshold 0.6, got %v", cfg.Settings.SimilarityThreshold)
	}
}

func TestLoad_DoesNotOverrideExistingSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"settings":{"log_file":"/custom/path.log","max_log_lines":42,"similarity_threshold":0.7}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Settings.LogFile != "/custom/path.log" {
		t.Errorf("LogFile overwritten; got %q", cfg.Settings.LogFile)
	}
	if cfg.Settings.MaxLogLines != 42 {
		t.Errorf("MaxLogLines overwritten; got %d", cfg.Settings.MaxLogLines)
	}
	if cfg.Settings.SimilarityThreshold != 0.7 {
		t.Errorf("SimilarityThreshold overwritten; got %v", cfg.Settings.SimilarityThreshold)
	}
}

// ---------------------------------------------------------------------------
// LoadOrDefault
// ---------------------------------------------------------------------------

func TestLoadOrDefault_FileExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"settings":{"log_file":"/tmp/log","max_log_lines":100,"similarity_threshold":0.7}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Settings.SimilarityThreshold != 0.7 {
		t.Errorf("expected SimilarityThreshold 0.7, got %v", cfg.Settings.SimilarityThreshold)
	}
}

func TestLoadOrDefault_FileNotFound_ReturnsDefault(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "missing.json")
	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil default config")
	}
	if cfg.Settings.MaxLogLines != 10000 {
		t.Errorf("expected default MaxLogLines 10000, got %d", cfg.Settings.MaxLogLines)
	}
	if cfg.Settings.SimilarityThreshold != 0.6 {
		t.Errorf("expected default SimilarityThreshold 0.6, got %v", cfg.Settings.SimilarityThreshold)
	}
}

func TestLoadOrDefault_BadJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("BAD"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadOrDefault(path)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

// TestLoadOrDefault_OtherReadError exercises the non-IsNotExist error branch.
// We use a directory path as the "file" which causes os.ReadFile to fail
// with an error that is NOT os.IsNotExist.
func TestLoadOrDefault_OtherReadError(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	_, err := LoadOrDefault(dirPath)
	if err == nil {
		t.Fatal("expected error when reading directory as file, got nil")
	}
	if os.IsNotExist(err) {
		t.Errorf("error should NOT be IsNotExist; got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestSave_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	cfg := &Config{
		Settings: Settings{
			LogFile:             "/tmp/log.log",
			MaxLogLines:         100,
			SimilarityThreshold: 0.7,
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save returned error: %v", err)
	}
	if loaded.Settings.LogFile != "/tmp/log.log" {
		t.Errorf("unexpected LogFile after round-trip: %q", loaded.Settings.LogFile)
	}
	if loaded.Settings.SimilarityThreshold != 0.7 {
		t.Errorf("unexpected SimilarityThreshold after round-trip: %v", loaded.Settings.SimilarityThreshold)
	}
}

func TestSave_MkdirFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create a regular file where Save expects a directory parent.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// Try to save inside a path that treats "blocker" as a directory.
	path := filepath.Join(blocker, "config.json")
	err := Save(path, &Config{})
	if err == nil {
		t.Fatal("expected error when parent is a file, got nil")
	}
}

func TestSave_WriteFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create a directory at the exact file path so os.WriteFile fails.
	filePath := filepath.Join(dir, "config.json")
	if err := os.MkdirAll(filePath, 0755); err != nil {
		t.Fatal(err)
	}
	err := Save(filePath, &Config{})
	if err == nil {
		t.Fatal("expected error writing to a directory, got nil")
	}
}

// ---------------------------------------------------------------------------
// newDefault / applyDefaults (indirectly tested through LoadOrDefault)
// ---------------------------------------------------------------------------

func TestNewDefault_HasNonZeroDefaults(t *testing.T) {
	t.Parallel()
	cfg := newDefault()
	if cfg.Settings.MaxLogLines == 0 {
		t.Error("expected non-zero MaxLogLines from newDefault")
	}
	if cfg.Settings.LogFile == "" {
		t.Error("expected non-empty LogFile from newDefault")
	}
	if cfg.Settings.SimilarityThreshold == 0 {
		t.Error("expected non-zero SimilarityThreshold from newDefault")
	}
}

func TestApplyDefaults_OutOfRangeThreshold_UsesDefault(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.Settings.SimilarityThreshold = 1.5 // out of range
	cfg.applyDefaults()
	if cfg.Settings.SimilarityThreshold != 0.6 {
		t.Errorf("expected default 0.6 for out-of-range threshold, got %v", cfg.Settings.SimilarityThreshold)
	}
}

func TestApplyDefaults_ValidThreshold_Preserved(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.Settings.SimilarityThreshold = 0.8
	cfg.Settings.MaxLogLines = 100
	cfg.Settings.LogFile = "/tmp/test.log"
	cfg.applyDefaults()
	if cfg.Settings.SimilarityThreshold != 0.8 {
		t.Errorf("expected 0.8 preserved, got %v", cfg.Settings.SimilarityThreshold)
	}
}
