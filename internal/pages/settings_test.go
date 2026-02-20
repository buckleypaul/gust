package pages

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/config"
)

func TestSettingsArrowKeyNavigation(t *testing.T) {
	cfg := config.Defaults()
	p := NewSettingsPage(&cfg, t.TempDir())

	// Initial cursor at 0
	if p.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", p.cursor)
	}

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 1 {
		t.Fatalf("expected cursor=1 after down, got %d", p.cursor)
	}

	// Move down to last
	for i := 0; i < len(settingFields)-2; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if p.cursor != len(settingFields)-1 {
		t.Fatalf("expected cursor=%d at last field, got %d", len(settingFields)-1, p.cursor)
	}

	// Clamp: another down should not move past last
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != len(settingFields)-1 {
		t.Fatalf("expected cursor to clamp at %d, got %d", len(settingFields)-1, p.cursor)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != len(settingFields)-2 {
		t.Fatalf("expected cursor=%d after up, got %d", len(settingFields)-2, p.cursor)
	}

	// Clamp: move up past 0
	p.cursor = 0
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != 0 {
		t.Fatalf("expected cursor to clamp at 0, got %d", p.cursor)
	}
}

func TestSettingsEnterEditMode(t *testing.T) {
	cfg := config.Defaults()
	p := NewSettingsPage(&cfg, t.TempDir())

	if p.editing {
		t.Fatal("expected editing=false initially")
	}

	// Enter key activates editing
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !p.editing {
		t.Fatal("expected editing=true after Enter")
	}

	// Esc exits editing
	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.editing {
		t.Fatal("expected editing=false after Esc")
	}
}

func TestSettingsApplyBaudRate(t *testing.T) {
	cfg := config.Defaults()
	p := NewSettingsPage(&cfg, t.TempDir())

	// Navigate to the serial_baud_rate field (index 2)
	for p.cursor < 2 {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if settingFields[p.cursor].key != "serial_baud_rate" {
		t.Fatalf("expected cursor on serial_baud_rate, got %s", settingFields[p.cursor].key)
	}

	// Enter edit mode, type "9600", confirm
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Clear input then set value
	p.input.SetValue("9600")
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cfg.SerialBaudRate != 9600 {
		t.Fatalf("expected SerialBaudRate=9600, got %d", cfg.SerialBaudRate)
	}
}

func TestSettingsInvalidBaudRate(t *testing.T) {
	cfg := config.Defaults()
	originalBaud := cfg.SerialBaudRate
	p := NewSettingsPage(&cfg, t.TempDir())

	// Navigate to serial_baud_rate
	for p.cursor < 2 {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Enter edit mode, set invalid value
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p.input.SetValue("not-a-number")
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Baud rate should not change
	if cfg.SerialBaudRate != originalBaud {
		t.Fatalf("expected SerialBaudRate to remain %d, got %d", originalBaud, cfg.SerialBaudRate)
	}
	// Should not panic and should be done editing
	if p.editing {
		t.Fatal("expected editing=false after enter")
	}
}

func TestSettingsSaveUpdatesConfig(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk"
	p := NewSettingsPage(&cfg, wsRoot)

	// Press 's' to save
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if p.message == "" {
		t.Fatal("expected message after save")
	}

	// Verify file was written
	configPath := filepath.Join(wsRoot, ".gust", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("expected config file at %s, not found", configPath)
	}

	// Load and verify
	loaded := config.Load(wsRoot)
	if loaded.DefaultBoard != "nrf52840dk" {
		t.Fatalf("expected DefaultBoard=nrf52840dk, got %q", loaded.DefaultBoard)
	}
}
