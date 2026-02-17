# Project Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a dedicated Project page that centralizes project, board, shield, and Kconfig configuration, replacing ConfigPage and simplifying the Build page.

**Architecture:** Project page uses the sectioned form pattern from Build page: hardware inputs (project/board/shield) at top, Kconfig list below, board overlay read-only section at bottom. Board/shield selections broadcast to all pages via new message types. Build page simplified to show context labels and retain only pristine/cmake/execution.

**Tech Stack:** Bubble Tea, Lipgloss, existing Picker component, west CLI for board discovery, file I/O for Kconfig parsing.

---

## Task 1: Add Config Fields

**Files:**
- Modify: `internal/config/config.go:1-30`
- Modify: `internal/config/config_test.go`

**Step 1: Add LastShield field to Config struct**

Read current struct at `internal/config/config.go:15-23`. Add after `LastProject`:

```go
LastShield string `json:"last_shield,omitempty"`
```

**Step 2: Update mergeFromFile to handle LastShield**

After line 110 in `config.go`, add:

```go
if fileCfg.LastShield != "" {
    cfg.LastShield = fileCfg.LastShield
}
```

**Step 3: Write test for LastShield persistence**

Add to `internal/config/config_test.go` after `TestSaveAndLoad`:

```go
func TestLastShieldPersistence(t *testing.T) {
    tmp := t.TempDir()
    cfg := Config{
        LastProject: "samples/ble-beacon",
        LastShield:  "nrf7002ek",
    }

    err := Save(cfg, tmp, false)
    if err != nil {
        t.Fatalf("Save failed: %v", err)
    }

    loaded := Load(tmp)
    if loaded.LastShield != "nrf7002ek" {
        t.Errorf("expected LastShield=nrf7002ek, got=%s", loaded.LastShield)
    }
}
```

**Step 4: Run test**

```bash
cd /Users/paulbuckley/Projects/gust && go test ./internal/config -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/config/config.go internal/config/config_test.go && git commit -m "feat: add LastShield config field"
```

---

## Task 2: Add Message Types

**Files:**
- Modify: `internal/app/page.go:50-55`

**Step 1: Add BoardSelectedMsg and ShieldSelectedMsg**

After `ProjectSelectedMsg` struct (line 54), add:

```go
// BoardSelectedMsg is broadcast to all pages when board is selected.
type BoardSelectedMsg struct {
	Board string
}

// ShieldSelectedMsg is broadcast to all pages when shield is selected.
type ShieldSelectedMsg struct {
	Shield string
}
```

**Step 2: Verify no test failures**

```bash
cd /Users/paulbuckley/Projects/gust && go test ./internal/app -v
```

Expected: PASS (or no tests for page.go yet)

**Step 3: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/app/page.go && git commit -m "feat: add BoardSelectedMsg and ShieldSelectedMsg types"
```

---

## Task 3: Add ProjectPage PageID and Update PageOrder

**Files:**
- Modify: `internal/app/page.go:9-33`

**Step 1: Add ProjectPage to PageID enum**

After `WorkspacePage`, insert:

```go
ProjectPage
```

This shifts existing page IDs: BuildPage becomes index 2, etc.

**Step 2: Update PageOrder slice**

Replace lines 23-33 with:

```go
var PageOrder = []PageID{
	WorkspacePage,
	ProjectPage,
	BuildPage,
	FlashPage,
	MonitorPage,
	TestPage,
	ArtifactsPage,
	WestPage,
	SettingsPage,
}
```

Note: ConfigPage is removed (index was 7, now gone).

**Step 3: Verify compilation**

```bash
cd /Users/paulbuckley/Projects/gust && go build ./cmd/gust
```

Expected: Should show errors about ProjectPage not in page map (expected, we'll add it later)

**Step 4: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/app/page.go && git commit -m "feat: add ProjectPage to PageID enum and reorder sidebar"
```

---

## Task 4: Create Project Page Implementation

**Files:**
- Create: `internal/pages/project.go`

**Step 1: Write stub with imports and structure**

```go
package pages

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type formSection int

const (
	sectionHardware formSection = iota
	sectionKconfig
	sectionOverlay
)

type editMode int

const (
	editModeNone editMode = iota
	editModeKconfig
	editModeSearch
)

type ProjectPage struct {
	// Hardware fields
	projectPath  string
	boardInput   textinput.Model
	shieldInput  textinput.Model
	boards       []west.Board
	filteredBoards []west.Board
	boardCursor  int
	boardListOpen bool

	// Kconfig section
	kconfigEntries []kconfigEntry
	filteredKconfig []kconfigEntry
	kconfigCursor  int
	searchInput    textinput.Model
	editInput      textinput.Model
	editing        bool
	editingValue   bool

	// Overlay section
	overlayEntries []kconfigEntry
	overlayExists  bool

	// State
	focusedSection formSection
	wsRoot         string
	cfg            *config.Config
	width, height  int
	message        string
	loading        bool
}

func NewProjectPage(cfg *config.Config, wsRoot string) *ProjectPage {
	boardInput := textinput.New()
	boardInput.Placeholder = "type to search..."
	boardInput.CharLimit = 128
	boardInput.Prompt = ""
	if cfg.DefaultBoard != "" {
		boardInput.SetValue(cfg.DefaultBoard)
	}

	shieldInput := textinput.New()
	shieldInput.Placeholder = "e.g. nrf7002ek"
	shieldInput.CharLimit = 128
	shieldInput.Prompt = ""
	if cfg.LastShield != "" {
		shieldInput.SetValue(cfg.LastShield)
	}

	searchInput := textinput.New()
	searchInput.Placeholder = "Search symbols..."
	searchInput.CharLimit = 64

	editInput := textinput.New()
	editInput.CharLimit = 256

	boardInput.Focus()

	return &ProjectPage{
		projectPath: cfg.LastProject,
		boardInput:  boardInput,
		shieldInput: shieldInput,
		searchInput: searchInput,
		editInput:   editInput,
		cfg:         cfg,
		wsRoot:      wsRoot,
	}
}

func (p *ProjectPage) Init() tea.Cmd {
	p.loading = true
	return tea.Batch(
		west.ListBoards(),
		p.loadKconfig,
	)
}

func (p *ProjectPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case app.ProjectSelectedMsg:
		p.projectPath = msg.Path
		p.cfg.LastProject = msg.Path
		config.Save(*p.cfg, p.wsRoot, false)
		return p, p.loadKconfig

	case west.BoardsLoadedMsg:
		p.loading = false
		if msg.Err != nil {
			p.message = fmt.Sprintf("Error loading boards: %v", msg.Err)
			return p, nil
		}
		p.boards = msg.Boards
		p.filterBoards()
		return p, nil

	case kconfigLoadedMsg:
		p.kconfigEntries = msg.entries
		p.filterKconfig()
		p.loadOverlay()
		return p, nil

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	return p, nil
}

func (p *ProjectPage) handleKey(msg tea.KeyMsg) (app.Page, tea.Cmd) {
	keyStr := msg.String()

	// Handle search mode
	if p.focusedSection == sectionKconfig && strings.HasPrefix(keyStr, "rune:") && p.searchInput.Focused() {
		switch keyStr {
		case "enter", "esc":
			p.searchInput.Blur()
			return p, nil
		}
		var cmd tea.Cmd
		p.searchInput, cmd = p.searchInput.Update(msg)
		p.filterKconfig()
		return p, cmd
	}

	// Handle edit mode
	if p.editing {
		switch keyStr {
		case "enter":
			p.editing = false
			p.editInput.Blur()
			// Save edited value
			if p.kconfigCursor < len(p.filteredKconfig) {
				entry := p.filteredKconfig[p.kconfigCursor]
				entry.Value = p.editInput.Value()
				p.filteredKconfig[p.kconfigCursor] = entry
				p.saveKconfig()
			}
			return p, nil
		case "esc":
			p.editing = false
			p.editInput.Blur()
			return p, nil
		}
		var cmd tea.Cmd
		p.editInput, cmd = p.editInput.Update(msg)
		return p, cmd
	}

	// Handle board dropdown
	if p.focusedSection == sectionHardware && p.boardListOpen {
		switch keyStr {
		case "up":
			if p.boardCursor > 0 {
				p.boardCursor--
			} else {
				p.boardListOpen = false
			}
			return p, nil
		case "down":
			if p.boardCursor < len(p.filteredBoards)-1 {
				p.boardCursor++
			}
			return p, nil
		case "enter":
			if len(p.filteredBoards) > 0 {
				p.boardInput.SetValue(p.filteredBoards[p.boardCursor].Name)
				p.boardListOpen = false
				p.cfg.DefaultBoard = p.filteredBoards[p.boardCursor].Name
				config.Save(*p.cfg, p.wsRoot, false)
				return p, func() tea.Msg { return app.BoardSelectedMsg{Board: p.cfg.DefaultBoard} }
			}
			return p, nil
		case "esc":
			p.boardListOpen = false
			return p, nil
		}
	}

	// Global keys
	switch keyStr {
	case "tab":
		p.advanceSection(1)
		return p, nil
	case "shift+tab":
		p.advanceSection(-1)
		return p, nil
	case "esc":
		p.blurAll()
		return p, nil
	}

	// Section-specific keys
	switch p.focusedSection {
	case sectionHardware:
		switch keyStr {
		case "p":
			return p, west.ListProjects(p.wsRoot, "")
		case "up":
			p.advanceSection(-1)
			return p, nil
		case "down":
			if p.focusedSection == sectionHardware {
				p.advanceSection(1)
				return p, nil
			}
		}
		// Board field-specific
		if p.boardInput.Focused() {
			switch keyStr {
			case "down":
				if len(p.filteredBoards) > 0 && !p.boardListOpen {
					p.boardListOpen = true
					p.boardCursor = 0
					return p, nil
				}
			case "enter":
				if len(p.filteredBoards) > 0 {
					p.boardInput.SetValue(p.filteredBoards[0].Name)
					p.cfg.DefaultBoard = p.filteredBoards[0].Name
					config.Save(*p.cfg, p.wsRoot, false)
					p.filterBoards()
					return p, func() tea.Msg { return app.BoardSelectedMsg{Board: p.cfg.DefaultBoard} }
				}
				return p, nil
			}
			var cmd tea.Cmd
			p.boardInput, cmd = p.boardInput.Update(msg)
			p.filterBoards()
			return p, cmd
		}
		// Shield field-specific
		if p.shieldInput.Focused() {
			switch keyStr {
			case "enter":
				p.cfg.LastShield = p.shieldInput.Value()
				config.Save(*p.cfg, p.wsRoot, false)
				return p, func() tea.Msg { return app.ShieldSelectedMsg{Shield: p.cfg.LastShield} }
			}
			var cmd tea.Cmd
			p.shieldInput, cmd = p.shieldInput.Update(msg)
			return p, cmd
		}

	case sectionKconfig:
		switch keyStr {
		case "/":
			p.searchInput.Focus()
			p.searchInput.SetValue("")
			return p, nil
		case "e":
			if p.kconfigCursor < len(p.filteredKconfig) {
				p.editing = true
				entry := p.filteredKconfig[p.kconfigCursor]
				p.editInput.SetValue(entry.Value)
				p.editInput.Focus()
			}
			return p, nil
		case "a":
			// Add new entry
			p.editInput.SetValue("")
			p.editInput.Focus()
			p.editing = true
			return p, nil
		case "d":
			// Delete entry
			if p.kconfigCursor < len(p.filteredKconfig) {
				entry := p.filteredKconfig[p.kconfigCursor]
				// Remove from actual entries
				for i, e := range p.kconfigEntries {
					if e.Name == entry.Name {
						p.kconfigEntries = append(p.kconfigEntries[:i], p.kconfigEntries[i+1:]...)
						break
					}
				}
				p.saveKconfig()
				p.filterKconfig()
			}
			return p, nil
		case "up":
			if p.kconfigCursor > 0 {
				p.kconfigCursor--
			}
			return p, nil
		case "down":
			if p.kconfigCursor < len(p.filteredKconfig)-1 {
				p.kconfigCursor++
			}
			return p, nil
		}
	}

	return p, nil
}

func (p *ProjectPage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Project"))
	b.WriteString("\n")

	if p.loading {
		b.WriteString("  Loading...")
		return b.String()
	}

	if p.message != "" {
		b.WriteString("  " + p.message + "\n\n")
	}

	inputWidth := p.width - 15 - 4
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Hardware section
	b.WriteString("  Project   ")
	b.WriteString(p.projectPath)
	b.WriteString("  [p]\n")

	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	boardLabel := "Board"
	if p.boardInput.Focused() {
		boardLabel = focusedLabel.Render(boardLabel)
	}
	b.WriteString("  " + boardLabel + "     ")
	p.boardInput.Width = inputWidth
	b.WriteString(p.boardInput.View())
	b.WriteString("\n")

	if p.focusedSection == sectionHardware && len(p.filteredBoards) > 0 && p.boardListOpen {
		dropdown := p.renderBoardDropdown(inputWidth)
		b.WriteString(dropdown)
	}

	shieldLabel := "Shield"
	if p.shieldInput.Focused() {
		shieldLabel = focusedLabel.Render(shieldLabel)
	}
	b.WriteString("  " + shieldLabel + "    ")
	p.shieldInput.Width = inputWidth
	b.WriteString(p.shieldInput.View())
	b.WriteString("\n\n")

	// Kconfig section
	b.WriteString("  ── Kconfig (prj.conf) ──\n")
	kconfigHeight := (p.height - 12) / 2
	if kconfigHeight < 3 {
		kconfigHeight = 3
	}
	b.WriteString(p.renderKconfigList(kconfigHeight))
	b.WriteString("\n")

	// Overlay section
	b.WriteString("  ── Board Overlay ──\n")
	b.WriteString(p.renderOverlay())
	b.WriteString("\n")

	b.WriteString(ui.DimStyle.Render("  tab: next  p: project  /: search  e: edit  a: add  d: delete"))

	return b.String()
}

func (p *ProjectPage) renderBoardDropdown(width int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", 14)

	count := len(p.filteredBoards)
	visible := count
	if visible > 10 {
		visible = 10
	}

	start := 0
	if p.boardCursor >= visible {
		start = p.boardCursor - visible + 1
	}
	end := start + visible
	if end > count {
		end = count
		start = end - visible
		if start < 0 {
			start = 0
		}
	}

	selectedStyle := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)

	for i := start; i < end; i++ {
		name := p.filteredBoards[i].Name
		if len(name) > width {
			name = name[:width]
		}
		prefix := "  "
		if p.boardListOpen && i == p.boardCursor {
			prefix = selectedStyle.Render("> ")
			name = selectedStyle.Render(name)
		} else {
			name = ui.DimStyle.Render(name)
		}
		b.WriteString(padding + prefix + name + "\n")
	}

	return b.String()
}

func (p *ProjectPage) renderKconfigList(height int) string {
	var b strings.Builder

	if len(p.filteredKconfig) == 0 {
		b.WriteString(ui.DimStyle.Render("  No Kconfig symbols"))
		return b.String()
	}

	start := p.kconfigCursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(p.filteredKconfig) {
		end = len(p.filteredKconfig)
		start = end - height
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		e := p.filteredKconfig[i]
		cursor := "  "
		if i == p.kconfigCursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%-30s = %s", cursor, e.Name, e.Value)
		if e.Comment != "" {
			line += ui.DimStyle.Render("  # " + e.Comment)
		}
		b.WriteString("  " + line + "\n")
	}

	return b.String()
}

func (p *ProjectPage) renderOverlay() string {
	var b strings.Builder

	board := p.boardInput.Value()
	if board == "" {
		b.WriteString("  (no board selected)\n")
		return b.String()
	}

	if len(p.overlayEntries) == 0 && !p.overlayExists {
		b.WriteString(ui.DimStyle.Render("  (no overlay for this board)"))
		return b.String()
	}

	if len(p.overlayEntries) > 0 {
		confFile := fmt.Sprintf("  boards/%s.conf:\n", board)
		b.WriteString(confFile)
		for _, e := range p.overlayEntries {
			line := fmt.Sprintf("    %s = %s\n", e.Name, e.Value)
			b.WriteString(line)
		}
	}

	if p.overlayExists {
		overlayFile := fmt.Sprintf("  boards/%s.overlay: (exists)\n", board)
		b.WriteString(overlayFile)
	}

	return b.String()
}

func (p *ProjectPage) Name() string { return "Project" }

func (p *ProjectPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "select project")),
	}
}

func (p *ProjectPage) InputCaptured() bool {
	return p.boardInput.Focused() || p.shieldInput.Focused() || p.editing || p.searchInput.Focused()
}

func (p *ProjectPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// Helper functions

func (p *ProjectPage) advanceSection(dir int) {
	p.focusedSection = formSection((int(p.focusedSection) + 3 + dir) % 3)
}

func (p *ProjectPage) blurAll() {
	p.boardInput.Blur()
	p.shieldInput.Blur()
	p.searchInput.Blur()
	p.editInput.Blur()
	p.boardListOpen = false
}

func (p *ProjectPage) filterBoards() {
	query := strings.ToLower(p.boardInput.Value())
	if query == "" {
		p.filteredBoards = p.boards
	} else {
		p.filteredBoards = nil
		for _, b := range p.boards {
			if strings.Contains(strings.ToLower(b.Name), query) {
				p.filteredBoards = append(p.filteredBoards, b)
			}
		}
	}
	if p.boardCursor >= len(p.filteredBoards) {
		p.boardCursor = len(p.filteredBoards) - 1
	}
	if p.boardCursor < 0 {
		p.boardCursor = 0
	}
}

func (p *ProjectPage) loadKconfig() tea.Msg {
	if p.projectPath == "" {
		return kconfigLoadedMsg{entries: nil, err: nil}
	}
	confPath := filepath.Join(p.wsRoot, p.projectPath, "prj.conf")
	entries, err := parsePrjConf(confPath)
	return kconfigLoadedMsg{entries: entries, err: err}
}

func (p *ProjectPage) filterKconfig() {
	query := strings.ToLower(p.searchInput.Value())
	if query == "" {
		p.filteredKconfig = p.kconfigEntries
	} else {
		p.filteredKconfig = nil
		for _, e := range p.kconfigEntries {
			if strings.Contains(strings.ToLower(e.Name), query) ||
				strings.Contains(strings.ToLower(e.Value), query) {
				p.filteredKconfig = append(p.filteredKconfig, e)
			}
		}
	}
	if p.kconfigCursor >= len(p.filteredKconfig) {
		p.kconfigCursor = len(p.filteredKconfig) - 1
	}
	if p.kconfigCursor < 0 {
		p.kconfigCursor = 0
	}
}

func (p *ProjectPage) loadOverlay() {
	board := p.boardInput.Value()
	if board == "" {
		p.overlayEntries = nil
		p.overlayExists = false
		return
	}

	// Check .conf file
	confPath := filepath.Join(p.wsRoot, p.projectPath, "boards", board+".conf")
	entries, _ := parsePrjConf(confPath)
	p.overlayEntries = entries

	// Check .overlay file
	overlayPath := filepath.Join(p.wsRoot, p.projectPath, "boards", board+".overlay")
	_, err := os.Stat(overlayPath)
	p.overlayExists = err == nil
}

func (p *ProjectPage) saveKconfig() {
	if p.projectPath == "" {
		return
	}
	confPath := filepath.Join(p.wsRoot, p.projectPath, "prj.conf")
	f, err := os.Create(confPath)
	if err != nil {
		p.message = fmt.Sprintf("Error saving: %v", err)
		return
	}
	defer f.Close()

	for _, e := range p.kconfigEntries {
		line := fmt.Sprintf("%s=%s", e.Name, e.Value)
		if e.Comment != "" {
			line += " # " + e.Comment
		}
		f.WriteString(line + "\n")
	}
	p.message = "Saved"
}
```

Note: Add missing imports at top of file:
```go
"github.com/charmbracelet/lipgloss"
```

**Step 2: Run basic build check**

```bash
cd /Users/paulbuckley/Projects/gust && go build ./cmd/gust 2>&1 | head -20
```

Expected: Will show errors about app.ProjectPage not being in pageMap (expected)

**Step 3: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/pages/project.go && git commit -m "feat: create Project page with hardware, kconfig, overlay sections"
```

---

## Task 5: Update Build Page - Remove Hardware Fields

**Files:**
- Modify: `internal/pages/build.go:27-131`

**Step 1: Remove project/board/shield field enum values**

Delete lines 29-34 (fieldProject, fieldBoard, fieldShield). Update fieldPristine and fieldCMakeArgs to account for removed fields. Resulting enum should be:

```go
const (
	fieldPristine formField = iota
	fieldCMakeArgs
	fieldCount
)
```

**Step 2: Remove input fields from BuildPage struct**

Delete lines 54-57 (projectInput, boardInput, shieldInput) and board-related fields (lines 61-65).

Struct should retain only:
- cmakeInput
- pristine
- boards/filteredBoards (keep for backward compat if needed, or remove)
- state, output, viewport, store, cfg, wsRoot, cwd, selectedBoard, buildStart, width, height, message, loading

**Step 3: Update NewBuildPage constructor**

Remove projectInput, boardInput, shieldInput initialization. Initialize only cmakeInput.

**Step 4: Update Init()**

Keep west.ListBoards() call.

**Step 5: Handle message flow**

Update Update() to receive and store BoardSelectedMsg and ShieldSelectedMsg:

```go
case app.BoardSelectedMsg:
    p.selectedBoard = msg.Board
    return p, nil

case app.ShieldSelectedMsg:
    // Store for display/use in build command
    return p, nil
```

Add a shieldInput field to struct for this.

**Step 6: Simplify viewForm**

Replace entire form rendering to show only:

```
Building: <projectPath>
Board: <board>  Shield: <shield>

Pristine  [ ]
CMake     [ input ]
```

**Step 7: Simplify handleKey**

Remove fieldProject, fieldBoard, fieldShield handling. Keep only fieldPristine and fieldCMakeArgs.

**Step 8: Run test to verify**

```bash
cd /Users/paulbuckley/Projects/gust && go test ./internal/pages -v 2>&1 | head -30
```

Expected: No compile errors

**Step 9: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/pages/build.go && git commit -m "refactor: remove hardware fields from Build page, keep only pristine and cmake args"
```

---

## Task 6: Update Top Bar - Show Board

**Files:**
- Modify: `internal/app/layout.go`

**Step 1: Update renderProjectBar to show board**

Current signature:

```go
func renderProjectBar(selectedProject string, width int, sidebarFocused bool) string
```

New signature:

```go
func renderProjectBar(selectedProject, selectedBoard string, width int, sidebarFocused bool) string
```

Inside function, change display:

```go
boardDisplay := selectedBoard
if boardDisplay == "" {
    boardDisplay = "(none)"
}
content := fmt.Sprintf("Project: %s  Board: %s", selectedProject, boardDisplay)
if sidebarFocused {
    content += "  [p] change"
}
```

**Step 2: Update renderLayout call sites**

In Model.View() in `internal/app/model.go`, update the renderProjectBar call to pass selectedBoard.

**Step 3: Run build check**

```bash
cd /Users/paulbuckley/Projects/gust && go build ./cmd/gust 2>&1 | head -20
```

Expected: May show error about selectedBoard not in Model (we'll add it next)

**Step 4: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/app/layout.go && git commit -m "feat: expand project bar to show both project and board"
```

---

## Task 7: Update App Model to Support BoardSelectedMsg and ShieldSelectedMsg

**Files:**
- Modify: `internal/app/model.go`

**Step 1: Add selectedBoard and selectedShield fields**

Find the Model struct and add:

```go
selectedBoard  string
selectedShield string
```

**Step 2: Initialize from config**

In Model.New() or appropriate init, set:

```go
m.selectedBoard = cfg.DefaultBoard
m.selectedShield = cfg.LastShield
```

**Step 3: Handle BoardSelectedMsg in Update()**

Add case:

```go
case app.BoardSelectedMsg:
    m.selectedBoard = msg.Board
    // Broadcast to all pages
    m.updateAllPages(msg)
```

**Step 4: Handle ShieldSelectedMsg in Update()**

Add case:

```go
case app.ShieldSelectedMsg:
    m.selectedShield = msg.Shield
    // Broadcast to all pages
    m.updateAllPages(msg)
```

**Step 5: Update renderProjectBar call**

In View(), change:

```go
projectBar := renderProjectBar(m.selectedProject, m.width, m.focusedPane == FocusSidebar)
```

to:

```go
projectBar := renderProjectBar(m.selectedProject, m.selectedBoard, m.width, m.focusedPane == FocusSidebar)
```

**Step 6: Run build check**

```bash
cd /Users/paulbuckley/Projects/gust && go build ./cmd/gust
```

Expected: PASS or clear errors to fix

**Step 7: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add internal/app/model.go && git commit -m "feat: add board and shield tracking to app Model"
```

---

## Task 8: Register Project Page in main.go

**Files:**
- Modify: `cmd/gust/main.go`

**Step 1: Create ProjectPage instance**

In the section where all pages are created (around where BuildPage, FlashPage, etc. are instantiated), add:

```go
projectPage := pages.NewProjectPage(&cfg, ws.Root)
```

**Step 2: Add to pageMap**

Add entry:

```go
pageMap[app.ProjectPage] = projectPage
```

**Step 3: Run full build**

```bash
cd /Users/paulbuckley/Projects/gust && go build ./cmd/gust
```

Expected: Should compile successfully

**Step 4: Commit**

```bash
cd /Users/paulbuckley/Projects/gust && git add cmd/gust/main.go && git commit -m "feat: register ProjectPage in main application"
```

---

## Task 9: Verification and Integration Test

**Files:**
- Test: Verify app runs and ProjectPage appears

**Step 1: Run the app**

```bash
cd /Users/paulbuckley/Projects/gust && make build
```

Expected: Binary compiles

**Step 2: Visual inspection**

Manually verify:
- Sidebar shows "Project" as second item after Workspace
- Top bar shows "Project: <path>  Board: <board>"
- Project page displays when selected
- Board dropdown works
- Kconfig list loads (if project has prj.conf)

**Step 3: Commit verification**

```bash
cd /Users/paulbuckley/Projects/gust && git log --oneline | head -10
```

Expected: All task commits appear

---

## Next Steps (Pending Implementation)

After these tasks complete:

1. Test ProjectPage keyboard navigation (tab cycling, board selection, kconfig editing)
2. Test message broadcasts (board/shield selection reaches other pages)
3. Test config persistence (selections saved/loaded across sessions)
4. Test Build page receives and displays board/shield context
5. Remove unused imports from build.go if any
6. Run full test suite: `go test ./...`
