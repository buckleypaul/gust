package pages

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfigLoadsEntries(t *testing.T) {
	wsRoot := t.TempDir()

	// Write a prj.conf
	if err := os.WriteFile(filepath.Join(wsRoot, "prj.conf"), []byte("CONFIG_FOO=y\nCONFIG_BAR=n\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := NewConfigPage(wsRoot)
	cmd := p.Init()
	if cmd == nil {
		t.Fatal("expected non-nil init cmd")
	}

	// Execute the load command and dispatch the result
	msg := cmd()
	loaded, ok := msg.(kconfigLoadedMsg)
	if !ok {
		t.Fatalf("expected kconfigLoadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("unexpected error: %v", loaded.err)
	}
	if len(loaded.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.entries))
	}
	if loaded.entries[0].Name != "CONFIG_FOO" || loaded.entries[0].Value != "y" {
		t.Fatalf("unexpected first entry: %+v", loaded.entries[0])
	}

	// Feed the message into Update
	p2, _ := p.Update(loaded)
	cp := p2.(*ConfigPage)
	if !cp.loaded {
		t.Fatal("expected loaded=true after kconfigLoadedMsg")
	}
	if len(cp.entries) != 2 {
		t.Fatalf("expected 2 entries in page after update, got %d", len(cp.entries))
	}
}

func TestConfigSearchFilter(t *testing.T) {
	wsRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(wsRoot, "prj.conf"), []byte("CONFIG_FOO=y\nCONFIG_BAR=n\nCONFIG_FOOBAR=m\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := NewConfigPage(wsRoot)
	msg := p.Init()()
	p.Update(msg)

	// Activate search with '/'
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !p.searching {
		t.Fatal("expected searching=true after '/'")
	}

	// Type "FOO" to filter
	p.search.SetValue("foo")
	p.filterEntries()

	for _, e := range p.filtered {
		if e.Name != "CONFIG_FOO" && e.Name != "CONFIG_FOOBAR" {
			t.Fatalf("unexpected entry in filtered results: %s", e.Name)
		}
	}
	if len(p.filtered) != 2 {
		t.Fatalf("expected 2 filtered entries, got %d: %v", len(p.filtered), p.filtered)
	}
}

func TestConfigCursorBounds(t *testing.T) {
	wsRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(wsRoot, "prj.conf"), []byte("CONFIG_A=y\nCONFIG_B=n\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := NewConfigPage(wsRoot)
	p.Update(p.Init()())

	// Move cursor to end
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", p.cursor)
	}

	// Clamp at last
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 1 {
		t.Fatalf("expected cursor to clamp at 1, got %d", p.cursor)
	}

	// Move back up and clamp at 0
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", p.cursor)
	}
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != 0 {
		t.Fatalf("expected cursor to clamp at 0, got %d", p.cursor)
	}
}

func TestConfigMissingFile(t *testing.T) {
	// wsRoot has no prj.conf
	p := NewConfigPage(t.TempDir())
	cmd := p.Init()
	msg := cmd()

	loaded, ok := msg.(kconfigLoadedMsg)
	if !ok {
		t.Fatalf("expected kconfigLoadedMsg, got %T", msg)
	}
	// Missing file is an error, but update should handle it gracefully
	page2, _ := p.Update(loaded)
	cp := page2.(*ConfigPage)
	if !cp.loaded {
		t.Fatal("expected loaded=true even when file missing")
	}
	// entries should be nil/empty
	if len(cp.entries) != 0 {
		t.Fatalf("expected 0 entries for missing file, got %d", len(cp.entries))
	}
	// Should not panic - view should render without issue
	_ = cp.View()
}
