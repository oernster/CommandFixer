// Package config handles loading, validating, and saving CommandFixer configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Settings holds global behaviour options.
type Settings struct {
	// LogFile is the path to the JSONL corrections log.
	LogFile string `json:"log_file"`
	// MaxLogLines is a soft cap for the log file (informational; rotation not yet implemented).
	MaxLogLines int `json:"max_log_lines"`
	// SimilarityThreshold controls how similar a typed subcommand must be to a
	// known subcommand before CommandFixer suggests a correction.
	// Valid range: (0.0, 1.0]. Lower values catch more typos but risk false
	// positives. Default: 0.6.
	SimilarityThreshold float64 `json:"similarity_threshold"`
}

// Config is the top-level configuration structure.
type Config struct {
	Settings Settings `json:"settings"`
}

// DefaultConfigDir returns the path to the directory that holds CommandFixer config.
// Default: $HOME/.typo-fixer
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".typo-fixer"), nil
}

// DefaultConfigPath returns the full path to the default config.json file.
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads and parses the JSON config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

// LoadOrDefault loads config from path.
// If the file does not exist, returns a default Config.
// Any other read or parse error is returned as-is.
func LoadOrDefault(path string) (*Config, error) {
	cfg, err := Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return newDefault(), nil
	}
	return cfg, err
}

// Save writes cfg to path as indented JSON, creating parent directories as needed.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

// newDefault returns a Config with sensible defaults.
func newDefault() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	return cfg
}

// applyDefaults fills in zero-value fields with reasonable defaults.
func (c *Config) applyDefaults() {
	if c.Settings.LogFile == "" {
		home, _ := os.UserHomeDir()
		c.Settings.LogFile = filepath.Join(home, ".typo-fixer", "corrections.log")
	}
	if c.Settings.MaxLogLines == 0 {
		c.Settings.MaxLogLines = 10000
	}
	if c.Settings.SimilarityThreshold <= 0 || c.Settings.SimilarityThreshold > 1 {
		c.Settings.SimilarityThreshold = 0.6
	}
}
