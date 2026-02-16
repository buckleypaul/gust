package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.BuildDir != "build" {
		t.Errorf("expected BuildDir=build, got=%s", cfg.BuildDir)
	}
	if cfg.SerialBaudRate != 115200 {
		t.Errorf("expected SerialBaudRate=115200, got=%d", cfg.SerialBaudRate)
	}
}

func TestLoadMerge(t *testing.T) {
	// Create a workspace config
	tmp := t.TempDir()
	gustDir := filepath.Join(tmp, ".gust")
	os.MkdirAll(gustDir, 0o755)
	os.WriteFile(filepath.Join(gustDir, "config.json"), []byte(`{
		"default_board": "nrf52840dk_nrf52840",
		"serial_baud_rate": 9600
	}`), 0o644)

	cfg := Load(tmp)

	if cfg.DefaultBoard != "nrf52840dk_nrf52840" {
		t.Errorf("expected default_board from workspace, got=%s", cfg.DefaultBoard)
	}
	if cfg.SerialBaudRate != 9600 {
		t.Errorf("expected baud rate 9600 from workspace, got=%d", cfg.SerialBaudRate)
	}
	// BuildDir should still be default since not overridden
	if cfg.BuildDir != "build" {
		t.Errorf("expected default BuildDir=build, got=%s", cfg.BuildDir)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	cfg := Config{
		DefaultBoard:   "esp32",
		BuildDir:       "mybuild",
		SerialBaudRate: 57600,
	}

	err := Save(cfg, tmp, false)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmp, ".gust", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back
	loaded := Load(tmp)
	if loaded.DefaultBoard != "esp32" {
		t.Errorf("expected DefaultBoard=esp32, got=%s", loaded.DefaultBoard)
	}
	if loaded.BuildDir != "mybuild" {
		t.Errorf("expected BuildDir=mybuild, got=%s", loaded.BuildDir)
	}
	if loaded.SerialBaudRate != 57600 {
		t.Errorf("expected SerialBaudRate=57600, got=%d", loaded.SerialBaudRate)
	}
}
