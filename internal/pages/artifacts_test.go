package pages

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/store"
)

func TestArtifactsTabSwitchRight(t *testing.T) {
	p := NewArtifactsPage(store.New(t.TempDir()))

	if p.activeTab != tabBuilds {
		t.Fatalf("expected initial tab=tabBuilds(0), got %d", p.activeTab)
	}

	// Advance through all tabs and wrap
	p.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.activeTab != tabFlashes {
		t.Fatalf("expected tabFlashes(1), got %d", p.activeTab)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.activeTab != tabTests {
		t.Fatalf("expected tabTests(2), got %d", p.activeTab)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.activeTab != tabSerialLogs {
		t.Fatalf("expected tabSerialLogs(3), got %d", p.activeTab)
	}

	// Wrap at last tab
	p.Update(tea.KeyMsg{Type: tea.KeyRight})
	if p.activeTab != tabBuilds {
		t.Fatalf("expected wrap back to tabBuilds(0), got %d", p.activeTab)
	}
}

func TestArtifactsTabSwitchLeft(t *testing.T) {
	p := NewArtifactsPage(store.New(t.TempDir()))

	// Wrap at first tab
	p.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.activeTab != tabSerialLogs {
		t.Fatalf("expected wrap to tabSerialLogs(%d), got %d", tabSerialLogs, p.activeTab)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p.activeTab != tabTests {
		t.Fatalf("expected tabTests(%d), got %d", tabTests, p.activeTab)
	}
}

func TestArtifactsBuildsTabRendersRecords(t *testing.T) {
	st := store.New(t.TempDir())
	if err := st.AddBuild(store.BuildRecord{
		Board:     "nrf52840dk",
		App:       "samples/hello_world",
		Timestamp: time.Now(),
		Success:   true,
		Duration:  "12s",
	}); err != nil {
		t.Fatalf("AddBuild: %v", err)
	}

	p := NewArtifactsPage(st)
	p.SetSize(120, 40)
	output := p.View()

	if !strings.Contains(output, "nrf52840dk") {
		t.Fatalf("expected board name in view, got:\n%s", output)
	}
}

func TestArtifactsEmptyStore(t *testing.T) {
	p := NewArtifactsPage(store.New(t.TempDir()))
	p.SetSize(120, 40)

	// Should not panic on any tab
	for i := 0; i < len(tabNames); i++ {
		_ = p.View()
		p.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
}
