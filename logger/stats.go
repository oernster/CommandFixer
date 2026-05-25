// Package logger records typo correction events and aggregates statistics.
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CorrectionEntry is a single correction event written to the JSONL log.
type CorrectionEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Original  string    `json:"original"`
	Corrected string    `json:"corrected"`
	Rule      string    `json:"rule"`
}

// Stats holds aggregated correction data loaded from the log file.
type Stats struct {
	TotalCorrections int
	RuleCounts       map[string]int
	History          []CorrectionEntry
}

// Logger appends CorrectionEntry records to a JSONL log file.
// Safe for concurrent use.
type Logger struct {
	mu      sync.Mutex
	logPath string
}

// New creates a Logger that writes to logPath.
func New(logPath string) *Logger {
	return &Logger{logPath: logPath}
}

// Log appends one correction event to the log file.
// Creates the parent directory if it does not exist.
func (l *Logger) Log(original, corrected, rule string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := CorrectionEntry{
		Timestamp: time.Now().UTC(),
		Original:  original,
		Corrected: corrected,
		Rule:      rule,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(l.logPath), 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// ReadStats loads all entries from logPath and returns aggregated Stats.
// If logPath does not exist, empty Stats are returned without error.
func ReadStats(logPath string) (*Stats, error) {
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return emptyStats(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read log %q: %w", logPath, err)
	}
	stats := emptyStats()
	for _, line := range splitLines(string(data)) {
		var entry CorrectionEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines gracefully
		}
		stats.TotalCorrections++
		stats.RuleCounts[entry.Rule]++
		stats.History = append(stats.History, entry)
	}
	return stats, nil
}

func emptyStats() *Stats {
	return &Stats{RuleCounts: make(map[string]int)}
}

// splitLines returns non-empty, non-whitespace-only lines from s.
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
