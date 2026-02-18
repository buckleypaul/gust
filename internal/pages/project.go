package pages

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

// projField identifies the currently focused field on the Project page.
type projField int

const (
	projFieldProject projField = iota
	projFieldBoard
	projFieldShield
	projFieldBuildDir
	projFieldRunner
	projFieldKconfig
	projFieldCMake  // cmake args input in build section
	projFieldCount
)

// ProjectPage owns project/board/shield selection, Kconfig editing, and board
// overlay display.
type ProjectPage struct {
	// Dependencies
	cfg          *config.Config
	wsRoot       string
	manifestPath string

	// Hardware section
	projectInput  textinput.Model
	boardInput    textinput.Model
	shieldInput   textinput.Model
	buildDirInput textinput.Model
	runnerInput   textinput.Model

	// Project type-ahead
	projects         []west.Project
	filteredProjects []west.Project
	projectListOpen  bool
	projectCursor    int
	projectPath      string // confirmed selection

	// Board type-ahead
	boards         []west.Board
	filteredBoards []west.Board
	boardCursor    int
	boardListOpen  bool

	// Focused field
	focusedField projField

	// Kconfig section
	kconfigEntries  []kconfigEntry
	kconfigFiltered []kconfigEntry
	kconfigCursor   int
	kconfigLoaded   bool

	// Search mode
	searchInput textinput.Model

	// Edit mode
	editInput textinput.Model
	editing   bool

	// Add mode
	addInput textinput.Model
	adding   bool

	// Overlay section
	overlayEntries []kconfigEntry
	overlayExists  bool

	// Build/flash sub-components
	build  buildSection
	flash  flashSection
	store  *store.Store
	runner west.Runner

	// Shared output panel
	output          strings.Builder
	viewport        viewport.Model
	activeOp        string // "Build" or "Flash"
	activeRequestID string

	// Metadata
	width, height int
	message       string
	loading       bool
}

// NewProjectPage creates a new ProjectPage.
func NewProjectPage(s *store.Store, cfg *config.Config, wsRoot string, manifestPath string, runners ...west.Runner) *ProjectPage {
	project := textinput.New()
	project.Placeholder = "type to search..."
	project.CharLimit = 256
	project.Prompt = ""
	if cfg.LastProject != "" {
		project.SetValue(cfg.LastProject)
	}

	board := textinput.New()
	board.Placeholder = "type to search..."
	board.CharLimit = 128
	board.Prompt = ""
	if cfg.DefaultBoard != "" {
		board.SetValue(cfg.DefaultBoard)
	}

	shield := textinput.New()
	shield.Placeholder = "e.g. nrf7002ek"
	shield.CharLimit = 128
	shield.Prompt = ""
	if cfg.LastShield != "" {
		shield.SetValue(cfg.LastShield)
	}

	buildDir := textinput.New()
	buildDir.Placeholder = "e.g. build-custom"
	buildDir.CharLimit = 256
	buildDir.Prompt = ""
	if cfg.BuildDir != "" {
		buildDir.SetValue(cfg.BuildDir)
	}

	runnerIn := textinput.New()
	runnerIn.Placeholder = "e.g. jlink, openocd"
	runnerIn.CharLimit = 128
	runnerIn.Prompt = ""
	if cfg.FlashRunner != "" {
		runnerIn.SetValue(cfg.FlashRunner)
	}

	search := textinput.New()
	search.Placeholder = "Search symbols..."
	search.CharLimit = 64
	search.Prompt = ""

	edit := textinput.New()
	edit.Placeholder = "new value"
	edit.CharLimit = 256
	edit.Prompt = ""

	add := textinput.New()
	add.Placeholder = "CONFIG_FOO=y"
	add.CharLimit = 256
	add.Prompt = ""

	runner := west.RealRunner()
	if len(runners) > 0 && runners[0] != nil {
		runner = runners[0]
	}

	p := &ProjectPage{
		cfg:           cfg,
		wsRoot:        wsRoot,
		manifestPath:  manifestPath,
		projectInput:  project,
		boardInput:    board,
		shieldInput:   shield,
		buildDirInput: buildDir,
		runnerInput:   runnerIn,
		searchInput:   search,
		editInput:     edit,
		addInput:      add,
		projectPath:   cfg.LastProject,
		focusedField:  projFieldProject,
		store:         s,
		runner:        runner,
		build:         newBuildSection(),
		viewport:      viewport.New(0, 0),
	}

	project.Focus()
	p.projectInput = project

	return p
}

func (p *ProjectPage) Init() tea.Cmd {
	p.loading = true
	return tea.Batch(
		west.ListBoards(),
		west.ListProjects(p.wsRoot, p.manifestPath),
		p.loadKconfig,
	)
}

func (p *ProjectPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case app.ProjectSelectedMsg:
		// Only reload kconfig if the path actually changed (avoid double load
		// when we broadcast our own selection and receive it back).
		if msg.Path != p.projectPath {
			p.projectPath = msg.Path
			p.projectInput.SetValue(msg.Path)
			p.filterProjects()
			p.cfg.LastProject = msg.Path
			p.kconfigLoaded = false
			return p, p.loadKconfig
		}
		return p, nil

	case app.BoardSelectedMsg:
		p.boardInput.SetValue(msg.Board)
		p.filterBoards()
		p.loadOverlay()
		return p, nil

	case app.ShieldSelectedMsg:
		p.shieldInput.SetValue(msg.Shield)
		return p, nil

	case app.BuildDirChangedMsg:
		p.buildDirInput.SetValue(msg.Dir)
		return p, nil

	case app.FlashRunnerChangedMsg:
		p.runnerInput.SetValue(msg.Runner)
		return p, nil

	case west.BoardsLoadedMsg:
		p.loading = false
		if msg.Err != nil {
			p.message = fmt.Sprintf("Error loading boards: %v", msg.Err)
			return p, nil
		}
		p.boards = msg.Boards
		p.filterBoards()
		return p, nil

	case west.ProjectsLoadedMsg:
		if msg.Err == nil {
			p.projects = msg.Projects
			p.filterProjects()
		}
		return p, nil

	case kconfigLoadedMsg:
		p.kconfigLoaded = true
		if msg.err != nil {
			// prj.conf may not exist yet - that's fine
			p.kconfigEntries = nil
			p.kconfigFiltered = nil
		} else {
			p.kconfigEntries = msg.entries
			p.filterKconfig()
		}
		p.loadOverlay()
		return p, nil

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	return p, nil
}

func (p *ProjectPage) handleKey(msg tea.KeyMsg) (app.Page, tea.Cmd) {
	keyStr := msg.String()

	// Add mode: forward to addInput, intercept enter/esc
	if p.adding {
		switch keyStr {
		case "enter":
			raw := strings.TrimSpace(p.addInput.Value())
			if raw != "" {
				parts := strings.SplitN(raw, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					p.kconfigEntries = append(p.kconfigEntries, kconfigEntry{Name: name, Value: value})
					p.filterKconfig()
					if err := p.saveKconfig(); err != nil {
						p.message = fmt.Sprintf("Failed to save prj.conf: %v", err)
					}
				}
			}
			p.adding = false
			p.addInput.SetValue("")
			p.addInput.Blur()
			return p, nil
		case "esc":
			p.adding = false
			p.addInput.SetValue("")
			p.addInput.Blur()
			return p, nil
		}
		var cmd tea.Cmd
		p.addInput, cmd = p.addInput.Update(msg)
		return p, cmd
	}

	// Edit mode: forward to editInput, intercept enter/esc
	if p.editing {
		switch keyStr {
		case "enter":
			if p.kconfigCursor < len(p.kconfigFiltered) {
				newVal := p.editInput.Value()
				// Update in the full entries list
				name := p.kconfigFiltered[p.kconfigCursor].Name
				for i, e := range p.kconfigEntries {
					if e.Name == name {
						p.kconfigEntries[i].Value = newVal
						break
					}
				}
				p.filterKconfig()
				if err := p.saveKconfig(); err != nil {
					p.message = fmt.Sprintf("Failed to save prj.conf: %v", err)
				}
			}
			p.editing = false
			p.editInput.SetValue("")
			p.editInput.Blur()
			return p, nil
		case "esc":
			p.editing = false
			p.editInput.SetValue("")
			p.editInput.Blur()
			return p, nil
		}
		var cmd tea.Cmd
		p.editInput, cmd = p.editInput.Update(msg)
		return p, cmd
	}

	// Search mode: forward to searchInput, intercept enter/esc
	if p.searchInput.Focused() {
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

	// Project dropdown navigation when active
	if p.projectListOpen {
		switch keyStr {
		case "up":
			if p.projectCursor > 0 {
				p.projectCursor--
			} else {
				p.projectListOpen = false
			}
			return p, nil
		case "down":
			if p.projectCursor < len(p.filteredProjects)-1 {
				p.projectCursor++
			}
			return p, nil
		case "enter":
			if len(p.filteredProjects) > 0 {
				return p, p.selectProject(p.filteredProjects[p.projectCursor].Path)
			}
			return p, nil
		case "esc":
			p.projectListOpen = false
			return p, nil
		}
	}

	// Board dropdown navigation when active
	if p.boardListOpen {
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
				selected := p.filteredBoards[p.boardCursor].Name
				p.boardInput.SetValue(selected)
				p.boardListOpen = false
				p.filterBoards()
				p.loadOverlay()
				p.cfg.DefaultBoard = selected
				if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
					p.message = fmt.Sprintf("Board selected, but config save failed: %v", err)
				}
				// Broadcast board selection
				return p, func() tea.Msg {
					return app.BoardSelectedMsg{Board: selected}
				}
			}
			return p, nil
		case "esc":
			p.boardListOpen = false
			return p, nil
		}
	}

	// Global form keys
	switch keyStr {
	case "esc":
		p.projectListOpen = false
		p.boardListOpen = false
		p.blurAll()
		return p, nil
	}

	// Field-specific handling
	switch p.focusedField {
	case projFieldProject:
		switch keyStr {
		case "down":
			p.projectListOpen = false
			p.advanceField(1)
			return p, nil
		case "up":
			p.projectListOpen = false
			p.advanceField(-1)
			return p, nil
		case "enter":
			if len(p.filteredProjects) > 0 {
				return p, p.selectProject(p.filteredProjects[0].Path)
			}
			return p, nil
		}
		var cmd tea.Cmd
		p.projectInput, cmd = p.projectInput.Update(msg)
		p.filterProjects()
		return p, cmd

	case projFieldBoard:
		switch keyStr {
		case "down":
			p.boardListOpen = false
			p.advanceField(1)
			return p, nil
		case "up":
			p.boardListOpen = false
			p.advanceField(-1)
			return p, nil
		case "enter":
			if len(p.filteredBoards) > 0 {
				selected := p.filteredBoards[0].Name
				p.boardInput.SetValue(selected)
				p.filterBoards()
				p.loadOverlay()
				p.cfg.DefaultBoard = selected
				if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
					p.message = fmt.Sprintf("Board selected, but config save failed: %v", err)
				}
				return p, func() tea.Msg {
					return app.BoardSelectedMsg{Board: selected}
				}
			}
			return p, nil
		}
		var cmd tea.Cmd
		p.boardInput, cmd = p.boardInput.Update(msg)
		p.filterBoards()
		return p, cmd

	case projFieldShield:
		switch keyStr {
		case "enter":
			shield := p.shieldInput.Value()
			p.cfg.LastShield = shield
			if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
				p.message = fmt.Sprintf("Shield selected, but config save failed: %v", err)
			}
			return p, func() tea.Msg {
				return app.ShieldSelectedMsg{Shield: shield}
			}
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
		}
		var cmd tea.Cmd
		p.shieldInput, cmd = p.shieldInput.Update(msg)
		return p, cmd

	case projFieldBuildDir:
		switch keyStr {
		case "enter":
			dir := p.buildDirInput.Value()
			p.cfg.BuildDir = dir
			if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
				p.message = fmt.Sprintf("Build dir set, but config save failed: %v", err)
			}
			return p, func() tea.Msg {
				return app.BuildDirChangedMsg{Dir: dir}
			}
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
		}
		var cmd tea.Cmd
		p.buildDirInput, cmd = p.buildDirInput.Update(msg)
		return p, cmd

	case projFieldRunner:
		switch keyStr {
		case "enter":
			runner := p.runnerInput.Value()
			p.cfg.FlashRunner = runner
			if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
				p.message = fmt.Sprintf("Runner set, but config save failed: %v", err)
			}
			return p, func() tea.Msg {
				return app.FlashRunnerChangedMsg{Runner: runner}
			}
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
		}
		var cmd tea.Cmd
		p.runnerInput, cmd = p.runnerInput.Update(msg)
		return p, cmd

	case projFieldKconfig:
		switch keyStr {
		case "up":
			if p.kconfigCursor > 0 {
				p.kconfigCursor--
			} else {
				p.advanceField(-1)
			}
			return p, nil
		case "down":
			if p.kconfigCursor < len(p.kconfigFiltered)-1 {
				p.kconfigCursor++
			}
			return p, nil
		case "/":
			p.searchInput.Focus()
			return p, p.searchInput.Focus()
		case "e":
			if p.kconfigCursor < len(p.kconfigFiltered) && len(p.kconfigFiltered) > 0 {
				p.editing = true
				p.editInput.SetValue(p.kconfigFiltered[p.kconfigCursor].Value)
				p.editInput.Focus()
			}
			return p, nil
		case "a":
			p.adding = true
			p.addInput.SetValue("")
			p.addInput.Focus()
			return p, nil
		case "d":
			if p.kconfigCursor < len(p.kconfigFiltered) && len(p.kconfigFiltered) > 0 {
				name := p.kconfigFiltered[p.kconfigCursor].Name
				for i, e := range p.kconfigEntries {
					if e.Name == name {
						p.kconfigEntries = append(p.kconfigEntries[:i], p.kconfigEntries[i+1:]...)
						break
					}
				}
				p.filterKconfig()
				if err := p.saveKconfig(); err != nil {
					p.message = fmt.Sprintf("Failed to save prj.conf: %v", err)
				}
			}
			return p, nil
		}
	}

	return p, nil
}

// selectProject confirms a project selection, saves config, reloads kconfig,
// and broadcasts ProjectSelectedMsg.
func (p *ProjectPage) selectProject(path string) tea.Cmd {
	p.projectPath = path
	p.projectInput.SetValue(path)
	p.projectListOpen = false
	p.filterProjects()
	p.cfg.LastProject = path
	if err := config.Save(*p.cfg, p.wsRoot, false); err != nil {
		p.message = fmt.Sprintf("Project selected, but config save failed: %v", err)
	}
	p.kconfigLoaded = false
	return tea.Batch(
		p.loadKconfig,
		func() tea.Msg { return app.ProjectSelectedMsg{Path: path} },
	)
}

func (p *ProjectPage) advanceField(dir int) {
	p.blurCurrent()
	p.focusedField = projField((int(p.focusedField) + int(projFieldCount) + dir) % int(projFieldCount))
	if p.focusedField != projFieldProject {
		p.projectListOpen = false
	}
	if p.focusedField != projFieldBoard {
		p.boardListOpen = false
	}
	p.focusCurrent()
}

func (p *ProjectPage) blurAll() {
	p.projectInput.Blur()
	p.boardInput.Blur()
	p.shieldInput.Blur()
	p.buildDirInput.Blur()
	p.runnerInput.Blur()
	p.build.cmakeInput.Blur()
	p.projectListOpen = false
	p.boardListOpen = false
}

func (p *ProjectPage) blurCurrent() {
	switch p.focusedField {
	case projFieldProject:
		p.projectInput.Blur()
	case projFieldBoard:
		p.boardInput.Blur()
	case projFieldShield:
		p.shieldInput.Blur()
	case projFieldBuildDir:
		p.buildDirInput.Blur()
	case projFieldRunner:
		p.runnerInput.Blur()
	case projFieldCMake:
		p.build.cmakeInput.Blur()
	}
}

func (p *ProjectPage) focusCurrent() {
	switch p.focusedField {
	case projFieldProject:
		p.projectInput.Focus()
	case projFieldBoard:
		p.boardInput.Focus()
	case projFieldShield:
		p.shieldInput.Focus()
	case projFieldBuildDir:
		p.buildDirInput.Focus()
	case projFieldRunner:
		p.runnerInput.Focus()
	case projFieldCMake:
		p.build.cmakeInput.Focus()
	}
}

func (p *ProjectPage) View() string {
	var b strings.Builder

	if p.message != "" {
		b.WriteString("  " + p.message + "\n\n")
	}

	// Styles
	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	const lw = 9 // label width

	renderLabel := func(name string, field projField) string {
		padded := fmt.Sprintf("%-*s", lw, name)
		if p.focusedField == field {
			return focusedLabel.Render(padded)
		}
		return normalLabel.Render(padded)
	}

	inputWidth := p.width - lw - 6
	if inputWidth < 10 {
		inputWidth = 10
	}
	p.projectInput.Width = inputWidth
	p.boardInput.Width = inputWidth
	p.shieldInput.Width = inputWidth
	p.buildDirInput.Width = inputWidth
	p.runnerInput.Width = inputWidth

	// -- Hardware section --

	// Project input
	b.WriteString("  " + renderLabel("Project", projFieldProject) + " " + p.projectInput.View() + "\n")

	// Project dropdown
	if p.focusedField == projFieldProject && len(p.filteredProjects) > 0 {
		b.WriteString(p.renderProjectDropdown(inputWidth))
	}

	// Board input
	b.WriteString("  " + renderLabel("Board", projFieldBoard) + " " + p.boardInput.View() + "\n")

	// Board dropdown
	if p.focusedField == projFieldBoard && len(p.filteredBoards) > 0 {
		b.WriteString(p.renderBoardDropdown(inputWidth))
	}

	// Shield input
	b.WriteString("  " + renderLabel("Shield", projFieldShield) + " " + p.shieldInput.View() + "\n")

	// Build Dir input
	b.WriteString("  " + renderLabel("Build Dir", projFieldBuildDir) + " " + p.buildDirInput.View() + "\n")

	// Flash Runner input
	b.WriteString("  " + renderLabel("Runner", projFieldRunner) + " " + p.runnerInput.View() + "\n")

	b.WriteString("\n")

	// -- Kconfig section --
	sectionLabel := lipgloss.NewStyle().Foreground(ui.Subtle).Bold(true)
	separator := strings.Repeat("─", max(p.width-4, 10))
	b.WriteString("  " + sectionLabel.Render("── Kconfig (prj.conf) "+separator) + "\n")

	if !p.kconfigLoaded {
		b.WriteString("  " + ui.DimStyle.Render("Loading...") + "\n")
	} else if p.adding {
		b.WriteString("  " + ui.DimStyle.Render("New entry (name=value): ") + p.addInput.View() + "\n")
	} else if p.editing && p.kconfigCursor < len(p.kconfigFiltered) {
		entry := p.kconfigFiltered[p.kconfigCursor]
		b.WriteString("  " + ui.DimStyle.Render("Edit "+entry.Name+": ") + p.editInput.View() + "\n")
	} else if p.searchInput.Focused() {
		b.WriteString("  " + ui.DimStyle.Render("/") + p.searchInput.View() + "\n")
	}

	if p.kconfigLoaded && !p.adding && !p.editing {
		if len(p.kconfigFiltered) == 0 {
			if p.projectPath == "" {
				b.WriteString("  " + ui.DimStyle.Render("No project selected.") + "\n")
			} else {
				b.WriteString("  " + ui.DimStyle.Render("No Kconfig symbols found.") + "\n")
			}
		} else {
			listHeight := p.height - 18
			if listHeight < 3 {
				listHeight = 3
			}

			start := p.kconfigCursor - listHeight/2
			if start < 0 {
				start = 0
			}
			end := start + listHeight
			if end > len(p.kconfigFiltered) {
				end = len(p.kconfigFiltered)
				start = end - listHeight
				if start < 0 {
					start = 0
				}
			}

			for i := start; i < end; i++ {
				e := p.kconfigFiltered[i]
				cursor := "  "
				isFocused := p.focusedField == projFieldKconfig && i == p.kconfigCursor
				if isFocused {
					cursor = ui.BoldStyle.Render("> ")
				}
				line := fmt.Sprintf("%s%-40s = %s", cursor, e.Name, e.Value)
				if e.Comment != "" {
					line += ui.DimStyle.Render("  # " + e.Comment)
				}
				b.WriteString(line + "\n")
			}

			b.WriteString(fmt.Sprintf("\n  %d/%d symbols", p.kconfigCursor+1, len(p.kconfigFiltered)))
			if p.searchInput.Value() != "" {
				b.WriteString(fmt.Sprintf(" (filter: %s)", p.searchInput.Value()))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// -- Overlay section --
	board := p.boardInput.Value()
	boardFile := ""
	if board != "" {
		boardFile = strings.SplitN(board, "/", 2)[0]
	}
	overlayHeader := "── Board Overlay"
	if boardFile != "" {
		overlayHeader += " (" + boardFile + ")"
	}
	overlaySep := strings.Repeat("─", max(p.width-len(overlayHeader)-6, 10))
	b.WriteString("  " + sectionLabel.Render(overlayHeader+" "+overlaySep) + "\n")

	if len(p.overlayEntries) > 0 {
		b.WriteString("  " + ui.DimStyle.Render("boards/"+boardFile+".conf:") + "\n")
		for _, e := range p.overlayEntries {
			b.WriteString("    " + e.Name + "=" + e.Value + "\n")
		}
	} else if board != "" {
		b.WriteString("  " + ui.DimStyle.Render("No boards/"+boardFile+".conf") + "\n")
	}

	if p.overlayExists {
		b.WriteString("  " + ui.DimStyle.Render("boards/"+boardFile+".overlay: (exists)") + "\n")
	} else if board != "" {
		b.WriteString("  " + ui.DimStyle.Render("No boards/"+boardFile+".overlay") + "\n")
	}

	if board == "" {
		b.WriteString("  " + ui.DimStyle.Render("Select a board to see overlay.") + "\n")
	}

	b.WriteString("\n")

	// Help bar
	b.WriteString(ui.DimStyle.Render("  ↑/↓: navigate  /: search  e: edit  a: add  d: delete"))

	return b.String()
}

func (p *ProjectPage) renderProjectDropdown(width int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", 9+3) // lw + "  " prefix + " " after label

	count := len(p.filteredProjects)
	visible := count
	if visible > maxDropdownItems {
		visible = maxDropdownItems
	}

	start := 0
	if p.projectListOpen && p.projectCursor >= visible {
		start = p.projectCursor - visible + 1
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
		path := p.filteredProjects[i].Path
		if len(path) > width {
			path = path[:width]
		}
		prefix := "  "
		if p.projectListOpen && i == p.projectCursor {
			prefix = selectedStyle.Render("> ")
			path = selectedStyle.Render(path)
		} else {
			path = ui.DimStyle.Render(path)
		}
		b.WriteString(padding + prefix + path + "\n")
	}

	countStr := fmt.Sprintf("(%d/%d projects)", visible, count)
	b.WriteString(padding + "  " + ui.DimStyle.Render(countStr) + "\n")

	return b.String()
}

func (p *ProjectPage) renderBoardDropdown(width int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", 9+3) // lw + "  " prefix + " " after label

	count := len(p.filteredBoards)
	visible := count
	if visible > maxDropdownItems {
		visible = maxDropdownItems
	}

	start := 0
	if p.boardListOpen && p.boardCursor >= visible {
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

	countStr := fmt.Sprintf("(%d/%d boards)", visible, count)
	b.WriteString(padding + "  " + ui.DimStyle.Render(countStr) + "\n")

	return b.String()
}

func (p *ProjectPage) Name() string { return "Project" }

func (p *ProjectPage) ShortHelp() []key.Binding {
	if p.searchInput.Focused() {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "done")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	if p.editing {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	if p.adding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "add")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑/↓", "navigate")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	}
}

func (p *ProjectPage) InputCaptured() bool {
	return p.projectInput.Focused() || p.boardInput.Focused() || p.shieldInput.Focused() ||
		p.buildDirInput.Focused() || p.runnerInput.Focused() ||
		p.editing || p.adding || p.searchInput.Focused() || p.build.cmakeInput.Focused()
}

func (p *ProjectPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// loadKconfig reads prj.conf from the current project path.
func (p *ProjectPage) loadKconfig() tea.Msg {
	projectRoot := p.projectAbsPath()
	if projectRoot == "" {
		return kconfigLoadedMsg{entries: nil, err: nil}
	}
	confPath := filepath.Join(projectRoot, "prj.conf")
	entries, err := parsePrjConf(confPath)
	return kconfigLoadedMsg{entries: entries, err: err}
}

// loadOverlay reads the board-specific overlay files synchronously.
func (p *ProjectPage) loadOverlay() {
	board := p.boardInput.Value()
	projectRoot := p.projectAbsPath()
	if board == "" || projectRoot == "" {
		p.overlayEntries = nil
		p.overlayExists = false
		return
	}
	boardFile := strings.SplitN(board, "/", 2)[0]
	confPath := filepath.Join(projectRoot, "boards", boardFile+".conf")
	entries, _ := parsePrjConf(confPath)
	p.overlayEntries = entries

	overlayPath := filepath.Join(projectRoot, "boards", boardFile+".overlay")
	_, err := os.Stat(overlayPath)
	p.overlayExists = err == nil
}

// filterProjects narrows the project list based on the current input.
func (p *ProjectPage) filterProjects() {
	query := strings.ToLower(p.projectInput.Value())
	if query == "" {
		p.filteredProjects = p.projects
	} else {
		p.filteredProjects = nil
		for _, proj := range p.projects {
			if strings.Contains(strings.ToLower(proj.Path), query) {
				p.filteredProjects = append(p.filteredProjects, proj)
			}
		}
	}
	if p.projectCursor >= len(p.filteredProjects) {
		p.projectCursor = len(p.filteredProjects) - 1
	}
	if p.projectCursor < 0 {
		p.projectCursor = 0
	}
}

// filterBoards narrows the board list based on the current input.
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

// filterKconfig narrows the Kconfig list based on the search input.
func (p *ProjectPage) filterKconfig() {
	query := strings.ToLower(p.searchInput.Value())
	if query == "" {
		p.kconfigFiltered = p.kconfigEntries
	} else {
		p.kconfigFiltered = nil
		for _, e := range p.kconfigEntries {
			if strings.Contains(strings.ToLower(e.Name), query) ||
				strings.Contains(strings.ToLower(e.Value), query) {
				p.kconfigFiltered = append(p.kconfigFiltered, e)
			}
		}
	}
	if p.kconfigCursor >= len(p.kconfigFiltered) {
		p.kconfigCursor = len(p.kconfigFiltered) - 1
	}
	if p.kconfigCursor < 0 {
		p.kconfigCursor = 0
	}
}

// saveKconfig writes kconfigEntries back to prj.conf.
func (p *ProjectPage) saveKconfig() error {
	projectRoot := p.projectAbsPath()
	if projectRoot == "" {
		return nil
	}
	confPath := filepath.Join(projectRoot, "prj.conf")
	var lines []string
	for _, e := range p.kconfigEntries {
		line := e.Name + "=" + e.Value
		if e.Comment != "" {
			line += " # " + e.Comment
		}
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n") + "\n"
	return writeFileAtomic(confPath, []byte(content), 0o644)
}

func (p *ProjectPage) projectAbsPath() string {
	if p.projectPath == "" {
		return ""
	}
	if filepath.IsAbs(p.projectPath) {
		return p.projectPath
	}
	return filepath.Join(p.wsRoot, p.projectPath)
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

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
