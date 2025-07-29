// Package history provides coverage history tracking and comparison functionality
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrNoHistory indicates no coverage history is available
var ErrNoHistory = errors.New("no coverage history available")

// CoverageRecord represents a single coverage measurement
type CoverageRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	CommitSHA    string    `json:"commit_sha"`
	Branch       string    `json:"branch"`
	Percentage   float64   `json:"percentage"`
	TotalLines   int       `json:"total_lines"`
	CoveredLines int       `json:"covered_lines"`
}

// Manager manages coverage history storage and retrieval
type Manager struct {
	historyFile string
}

// NewManager creates a new coverage history manager
func NewManager(baseDir string) *Manager {
	historyFile := filepath.Join(baseDir, "coverage-history.json")
	return &Manager{
		historyFile: historyFile,
	}
}

// SaveRecord saves a coverage record to the history
func (m *Manager) SaveRecord(record *CoverageRecord) error {
	// Load existing history
	history, err := m.loadHistory()
	if err != nil {
		// If file doesn't exist, start with empty history
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load existing history: %w", err)
		}
		history = []CoverageRecord{}
	}

	// Add new record
	history = append(history, *record)

	// Keep only last 100 records to prevent unbounded growth
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	// Save updated history
	return m.saveHistory(history)
}

// GetLastRecord returns the most recent coverage record
func (m *Manager) GetLastRecord() (*CoverageRecord, error) {
	history, err := m.loadHistory()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoHistory
		}
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	if len(history) == 0 {
		return nil, ErrNoHistory
	}

	return &history[len(history)-1], nil
}

// GetChangeStatus compares current coverage with the last recorded coverage
func (m *Manager) GetChangeStatus(currentPercentage float64) (string, float64, error) {
	lastRecord, err := m.GetLastRecord()
	if err != nil {
		if errors.Is(err, ErrNoHistory) {
			return "stable", 0.0, nil
		}
		return "stable", 0.0, err
	}

	previousPercentage := lastRecord.Percentage
	diff := currentPercentage - previousPercentage

	// Define thresholds for change detection
	const threshold = 0.1 // 0.1% threshold

	if diff > threshold {
		return "improved", previousPercentage, nil
	} else if diff < -threshold {
		return "declined", previousPercentage, nil
	}

	return "stable", previousPercentage, nil
}

// loadHistory loads the coverage history from the JSON file
func (m *Manager) loadHistory() ([]CoverageRecord, error) {
	data, err := os.ReadFile(m.historyFile)
	if err != nil {
		return nil, err
	}

	var history []CoverageRecord
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}

	return history, nil
}

// ensureHistoryDir ensures the history directory exists
func (m *Manager) ensureHistoryDir() error {
	dirPath := filepath.Dir(m.historyFile)
	// Check if directory already exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Create directory with proper permissions
		return os.MkdirAll(dirPath, 0o750)
	}
	return nil
}

// saveHistory saves the coverage history to the JSON file
func (m *Manager) saveHistory(history []CoverageRecord) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Ensure the directory exists before writing
	if err := m.ensureHistoryDir(); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(m.historyFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}
