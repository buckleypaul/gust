package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultBaudRate = 115200
	DefaultBuildDir = "build"
)

// Config holds all gust configuration.
type Config struct {
	DefaultBoard   string `json:"default_board,omitempty"`
	BuildDir       string `json:"build_dir,omitempty"`
	SerialPort     string `json:"serial_port,omitempty"`
	SerialBaudRate int    `json:"serial_baud_rate,omitempty"`
	FlashRunner    string `json:"flash_runner,omitempty"`
	VenvPath       string `json:"venv_path,omitempty"`
}

// Defaults returns a Config with default values.
func Defaults() Config {
	return Config{
		BuildDir:       DefaultBuildDir,
		SerialBaudRate: DefaultBaudRate,
	}
}

// Load reads and merges global and workspace configs.
// Order: defaults → global (~/.config/gust/config.json) → workspace (.gust/config.json).
func Load(workspaceRoot string) Config {
	cfg := Defaults()

	// Global config
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "gust", "config.json")
		mergeFromFile(&cfg, globalPath)
	}

	// Workspace config
	if workspaceRoot != "" {
		wsPath := filepath.Join(workspaceRoot, ".gust", "config.json")
		mergeFromFile(&cfg, wsPath)
	}

	return cfg
}

// Save writes the config to the workspace .gust/config.json by default,
// or to the global config if global is true.
func Save(cfg Config, workspaceRoot string, global bool) error {
	var dir string
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(home, ".config", "gust")
	} else {
		dir = filepath.Join(workspaceRoot, ".gust")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)
}

func mergeFromFile(cfg *Config, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return
	}

	if fileCfg.DefaultBoard != "" {
		cfg.DefaultBoard = fileCfg.DefaultBoard
	}
	if fileCfg.BuildDir != "" {
		cfg.BuildDir = fileCfg.BuildDir
	}
	if fileCfg.SerialPort != "" {
		cfg.SerialPort = fileCfg.SerialPort
	}
	if fileCfg.SerialBaudRate != 0 {
		cfg.SerialBaudRate = fileCfg.SerialBaudRate
	}
	if fileCfg.FlashRunner != "" {
		cfg.FlashRunner = fileCfg.FlashRunner
	}
	if fileCfg.VenvPath != "" {
		cfg.VenvPath = fileCfg.VenvPath
	}
}
