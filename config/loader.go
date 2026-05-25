// Package config handles loading, validating, and saving CommandFixer configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TypoEntry represents a single typo-to-correction mapping.
// Set Regex to true to treat From as a regular expression pattern.
type TypoEntry struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Regex bool   `json:"regex,omitempty"`
}

// Settings holds global behaviour options.
type Settings struct {
	// LogFile is the path to the JSONL corrections log.
	LogFile string `json:"log_file"`
	// ShowCorrections controls whether the PS hook prints corrections.
	ShowCorrections bool `json:"show_corrections"`
	// MaxLogLines is a soft cap for the log file (informational; rotation not yet implemented).
	MaxLogLines int `json:"max_log_lines"`
}

// Config is the top-level configuration structure.
type Config struct {
	Typos    []TypoEntry `json:"typos"`
	Settings Settings    `json:"settings"`
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
// If the file does not exist, returns a default Config with no typos.
// Any other read or parse error is returned as-is.
func LoadOrDefault(path string) (*Config, error) {
	cfg, err := Load(path)
	if os.IsNotExist(err) {
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

// newDefault returns a Config with sensible defaults and no typo rules.
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
}
