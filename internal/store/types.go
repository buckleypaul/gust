package store

import "time"

// BuildRecord captures the result of a build operation.
type BuildRecord struct {
	Board     string    `json:"board"`
	App       string    `json:"app"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Duration  string    `json:"duration"`
	Artifacts []string  `json:"artifacts"`
	Shield    string    `json:"shield,omitempty"`
	Pristine  bool      `json:"pristine,omitempty"`
	CMakeArgs string    `json:"cmake_args,omitempty"`
}

// FlashRecord captures the result of a flash operation.
type FlashRecord struct {
	Board     string    `json:"board"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Duration  string    `json:"duration"`
}

// TestRecord captures the result of a test run.
type TestRecord struct {
	Board     string    `json:"board"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Duration  string    `json:"duration"`
	Output    string    `json:"output,omitempty"`
}

// SerialLog tracks a serial logging session.
type SerialLog struct {
	Port      string    `json:"port"`
	BaudRate  int       `json:"baud_rate"`
	Timestamp time.Time `json:"timestamp"`
	LogFile   string    `json:"log_file"`
}
