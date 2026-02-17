package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store manages persistence of build/flash/test records and serial logs.
type Store struct {
	root string
	mu   sync.Mutex
}

// New creates a Store rooted at the given directory (typically .gust/).
func New(root string) *Store {
	return &Store{root: root}
}

func (s *Store) historyDir() string {
	return filepath.Join(s.root, "history")
}

func (s *Store) logsDir() string {
	return filepath.Join(s.root, "logs")
}

// AddBuild appends a build record.
func (s *Store) AddBuild(r BuildRecord) error {
	return s.appendRecord("builds.json", r)
}

// AddFlash appends a flash record.
func (s *Store) AddFlash(r FlashRecord) error {
	return s.appendRecord("flashes.json", r)
}

// AddTest appends a test record.
func (s *Store) AddTest(r TestRecord) error {
	return s.appendRecord("tests.json", r)
}

// Builds returns all build records.
func (s *Store) Builds() ([]BuildRecord, error) {
	var records []BuildRecord
	err := s.loadRecords("builds.json", &records)
	return records, err
}

// Flashes returns all flash records.
func (s *Store) Flashes() ([]FlashRecord, error) {
	var records []FlashRecord
	err := s.loadRecords("flashes.json", &records)
	return records, err
}

// Tests returns all test records.
func (s *Store) Tests() ([]TestRecord, error) {
	var records []TestRecord
	err := s.loadRecords("tests.json", &records)
	return records, err
}

// SerialLogs returns all serial log entries.
func (s *Store) SerialLogs() ([]SerialLog, error) {
	var records []SerialLog
	err := s.loadRecords("serial_logs.json", &records)
	return records, err
}

// AddSerialLog appends a serial log entry.
func (s *Store) AddSerialLog(r SerialLog) error {
	return s.appendRecord("serial_logs.json", r)
}

// LogsDir returns the path to the logs directory, creating it if needed.
func (s *Store) LogsDir() (string, error) {
	dir := s.logsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *Store) appendRecord(filename string, record any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.historyDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(dir, filename)

	// Read existing records
	var records []json.RawMessage
	if data, err := os.ReadFile(path); err == nil {
		if len(data) > 0 {
			if err := json.Unmarshal(data, &records); err != nil {
				return fmt.Errorf("invalid %s: %w", path, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// Marshal and append new record
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	records = append(records, raw)

	// Write back
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o644)
}

func (s *Store) loadRecords(filename string, dest any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.historyDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, dest)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gust-tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}
