package pages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/west"
	tea "github.com/charmbracelet/bubbletea"
)

func TestProjectPageLoadKconfigUsesWorkspaceRelativeProjectPath(t *testing.T) {
	wsRoot := t.TempDir()
	projectDir := filepath.Join(wsRoot, "apps", "demo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	confPath := filepath.Join(projectDir, "prj.conf")
	if err := os.WriteFile(confPath, []byte("CONFIG_FOO=y\n"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cfg := config.Defaults()
	p := NewProjectPage(&cfg, wsRoot, "")
	p.projectPath = filepath.Join("apps", "demo")

	msg := p.loadKconfig()
	loaded, ok := msg.(kconfigLoadedMsg)
	if !ok {
		t.Fatalf("expected kconfigLoadedMsg, got %T", msg)
	}
	if loaded.err != nil {
		t.Fatalf("loadKconfig returned error: %v", loaded.err)
	}
	if len(loaded.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.entries))
	}
	if loaded.entries[0].Name != "CONFIG_FOO" || loaded.entries[0].Value != "y" {
		t.Fatalf("unexpected entry: %+v", loaded.entries[0])
	}
}

func TestProjectPageSaveKconfigWritesWorkspaceRelativeProjectPath(t *testing.T) {
	wsRoot := t.TempDir()
	projectDir := filepath.Join(wsRoot, "apps", "demo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	cfg := config.Defaults()
	p := NewProjectPage(&cfg, wsRoot, "")
	p.projectPath = filepath.Join("apps", "demo")
	p.kconfigEntries = []kconfigEntry{
		{Name: "CONFIG_BAR", Value: "42"},
	}

	if err := p.saveKconfig(); err != nil {
		t.Fatalf("saveKconfig failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "prj.conf"))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "CONFIG_BAR=42\n" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func TestProjectPageSearchFlowFiltersKconfigEntries(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	p := NewProjectPage(&cfg, wsRoot, "")
	p.focusedField = projFieldKconfig
	p.kconfigLoaded = true
	p.kconfigEntries = []kconfigEntry{
		{Name: "CONFIG_FOO", Value: "y"},
		{Name: "CONFIG_BAR", Value: "n"},
	}
	p.filterKconfig()

	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.searchInput.Focused() {
		t.Fatal("expected search input to be focused")
	}

	p = typeStringIntoProjectPage(p, "FOO")
	if len(p.kconfigFiltered) != 1 {
		t.Fatalf("expected 1 filtered entry, got %d", len(p.kconfigFiltered))
	}
	if p.kconfigFiltered[0].Name != "CONFIG_FOO" {
		t.Fatalf("expected CONFIG_FOO, got %s", p.kconfigFiltered[0].Name)
	}

	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyEnter})
	if p.searchInput.Focused() {
		t.Fatal("expected search input to be blurred after enter")
	}
}

func TestProjectPageAddEditDeleteFlow(t *testing.T) {
	wsRoot := t.TempDir()
	projectRel := filepath.Join("apps", "demo")
	projectDir := filepath.Join(wsRoot, projectRel)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	cfg := config.Defaults()
	p := NewProjectPage(&cfg, wsRoot, "")
	p.focusedField = projFieldKconfig
	p.projectPath = projectRel
	p.kconfigLoaded = true
	p.kconfigEntries = []kconfigEntry{{Name: "CONFIG_FOO", Value: "y"}}
	p.filterKconfig()

	// Add
	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !p.adding {
		t.Fatal("expected add mode")
	}
	p = typeStringIntoProjectPage(p, "CONFIG_NEW=y")
	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyEnter})
	if p.adding {
		t.Fatal("expected add mode to exit")
	}
	if !containsEntry(p.kconfigEntries, "CONFIG_NEW", "y") {
		t.Fatalf("expected CONFIG_NEW=y to be added: %+v", p.kconfigEntries)
	}

	// Edit
	p.kconfigCursor = findEntryIndex(p.kconfigFiltered, "CONFIG_NEW")
	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if !p.editing {
		t.Fatal("expected edit mode")
	}
	p.editInput.SetValue("n")
	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyEnter})
	if p.editing {
		t.Fatal("expected edit mode to exit")
	}
	if !containsEntry(p.kconfigEntries, "CONFIG_NEW", "n") {
		t.Fatalf("expected CONFIG_NEW=n after edit: %+v", p.kconfigEntries)
	}

	// Delete
	p.kconfigCursor = findEntryIndex(p.kconfigFiltered, "CONFIG_NEW")
	p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if containsName(p.kconfigEntries, "CONFIG_NEW") {
		t.Fatalf("expected CONFIG_NEW to be deleted: %+v", p.kconfigEntries)
	}

	// Verify persisted file reflects end state.
	data, err := os.ReadFile(filepath.Join(projectDir, "prj.conf"))
	if err != nil {
		t.Fatalf("read prj.conf failed: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "CONFIG_FOO=y") {
		t.Fatalf("expected CONFIG_FOO in prj.conf, got %q", text)
	}
	if strings.Contains(text, "CONFIG_NEW=") {
		t.Fatalf("did not expect CONFIG_NEW in prj.conf, got %q", text)
	}
}

func TestProjectPageBoardEnterSelection(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	p := NewProjectPage(&cfg, wsRoot, "")
	p.focusedField = projFieldBoard
	p.boards = []west.Board{{Name: "board-a"}, {Name: "board-b"}}
	p.filterBoards()

	// Enter on Board field selects the first filtered result
	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	p = page.(*ProjectPage)
	if cmd == nil {
		t.Fatal("expected selection command")
	}
	msg := cmd()
	boardMsg, ok := msg.(app.BoardSelectedMsg)
	if !ok {
		t.Fatalf("expected BoardSelectedMsg, got %T", msg)
	}
	if boardMsg.Board != "board-a" {
		t.Fatalf("expected board-a, got %s", boardMsg.Board)
	}
	if p.boardInput.Value() != "board-a" {
		t.Fatalf("expected board input to be set, got %q", p.boardInput.Value())
	}
}

func updateProjectPage(p *ProjectPage, msg tea.Msg) *ProjectPage {
	page, _ := p.Update(msg)
	return page.(*ProjectPage)
}

func typeStringIntoProjectPage(p *ProjectPage, value string) *ProjectPage {
	for _, r := range value {
		p = updateProjectPage(p, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return p
}

func containsEntry(entries []kconfigEntry, name, value string) bool {
	for _, e := range entries {
		if e.Name == name && e.Value == value {
			return true
		}
	}
	return false
}

func containsName(entries []kconfigEntry, name string) bool {
	for _, e := range entries {
		if e.Name == name {
			return true
		}
	}
	return false
}

func findEntryIndex(entries []kconfigEntry, name string) int {
	for i, e := range entries {
		if e.Name == name {
			return i
		}
	}
	return 0
}
