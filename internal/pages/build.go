package pages

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

type formField int

const (
	fieldProject formField = iota
	fieldBoard
	fieldShield
	fieldPristine
	fieldCMakeArgs
	fieldCount
)

type buildState int

const (
	buildStateIdle buildState = iota
	buildStateRunning
	buildStateDone
)

const (
	labelWidth       = 11 // "Project  " padded
	minLeftWidth     = 30
	maxLeftWidth     = 50
	maxDropdownItems = 10
)

type BuildPage struct {
	// Form inputs
	projectInput textinput.Model
	boardInput   textinput.Model
	shieldInput  textinput.Model
	cmakeInput   textinput.Model
	pristine     bool

	// Board type-ahead
	boards         []west.Board
	filteredBoards []west.Board
	boardCursor    int
	boardListOpen  bool

	// State
	focusedField formField
	state        buildState
	output       strings.Builder
	viewport     viewport.Model

	// Dependencies
	store  *store.Store
	cfg    *config.Config
	wsRoot string
	cwd    string

	// Metadata
	selectedBoard string
	buildStart    time.Time
	width, height int
	message       string
	loading       bool
}

func NewBuildPage(s *store.Store, cfg *config.Config, wsRoot string, cwd string) *BuildPage {
	project := textinput.New()
	project.Placeholder = "."
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

	cmake := textinput.New()
	cmake.Placeholder = "e.g. -DOVERLAY_CONFIG=overlay.conf"
	cmake.CharLimit = 512
	cmake.Prompt = ""

	vp := viewport.New(0, 0)

	// Focus first field
	project.Focus()

	return &BuildPage{
		projectInput: project,
		boardInput:   board,
		shieldInput:  shield,
		cmakeInput:   cmake,
		viewport:     vp,
		store:        s,
		cfg:          cfg,
		wsRoot:       wsRoot,
		cwd:          cwd,
		focusedField: fieldProject,
	}
}

func (p *BuildPage) Init() tea.Cmd {
	p.loading = true
	return west.ListBoards()
}

func (p *BuildPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case west.BoardsLoadedMsg:
		p.loading = false
		if msg.Err != nil {
			p.message = fmt.Sprintf("Error loading boards: %v", msg.Err)
			return p, nil
		}
		p.boards = msg.Boards
		p.filterBoards()
		return p, nil

	case west.CommandResultMsg:
		p.state = buildStateDone
		p.output.WriteString(msg.Output)
		success := msg.ExitCode == 0
		status := "success"
		if !success {
			status = fmt.Sprintf("failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\nBuild %s in %s\n", status, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		// Record build
		if p.store != nil {
			p.store.AddBuild(store.BuildRecord{
				Board:     p.selectedBoard,
				App:       p.projectValue(),
				Timestamp: p.buildStart,
				Success:   success,
				Duration:  msg.Duration.String(),
				Shield:    p.shieldInput.Value(),
				Pristine:  p.pristine,
				CMakeArgs: p.cmakeInput.Value(),
			})
		}

		// Persist last project and board to config
		if success {
			p.cfg.LastProject = p.projectValue()
			p.cfg.DefaultBoard = p.selectedBoard
			config.Save(*p.cfg, p.wsRoot, false)
		}
		return p, nil

	case tea.KeyMsg:
		return p.handleKey(msg)
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *BuildPage) handleKey(msg tea.KeyMsg) (app.Page, tea.Cmd) {
	// When running, only viewport scrolling and cancel
	if p.state == buildStateRunning {
		var cmd tea.Cmd
		p.viewport, cmd = p.viewport.Update(msg)
		return p, cmd
	}

	keyStr := msg.String()

	// Handle board dropdown navigation when active
	if p.boardListOpen {
		switch keyStr {
		case "up":
			if p.boardCursor > 0 {
				p.boardCursor--
			} else {
				// At top of list, return to input
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
				p.filterBoards()
			}
			return p, nil
		case "esc":
			p.boardListOpen = false
			return p, nil
		}
	}

	// Global form keys
	switch keyStr {
	case "tab":
		p.advanceField(1)
		return p, nil
	case "shift+tab":
		p.advanceField(-1)
		return p, nil
	case "ctrl+b":
		return p, p.startBuild()
	case "esc":
		if p.state == buildStateDone {
			p.state = buildStateIdle
			p.output.Reset()
			p.viewport.SetContent("")
			return p, nil
		}
		p.blurAll()
		return p, nil
	}

	// Field-specific handling
	switch p.focusedField {
	case fieldBoard:
		if keyStr == "down" && len(p.filteredBoards) > 0 {
			p.boardListOpen = true
			p.boardCursor = 0
			return p, nil
		}
		if keyStr == "enter" {
			// Select top match if available
			if len(p.filteredBoards) > 0 {
				p.boardInput.SetValue(p.filteredBoards[0].Name)
				p.filterBoards()
			}
			return p, nil
		}
		var cmd tea.Cmd
		p.boardInput, cmd = p.boardInput.Update(msg)
		p.filterBoards()
		return p, cmd

	case fieldPristine:
		if keyStr == "enter" || keyStr == " " {
			p.pristine = !p.pristine
			return p, nil
		}
		return p, nil

	case fieldProject:
		if keyStr == "enter" {
			return p, p.startBuild()
		}
		var cmd tea.Cmd
		p.projectInput, cmd = p.projectInput.Update(msg)
		return p, cmd

	case fieldShield:
		if keyStr == "enter" {
			return p, p.startBuild()
		}
		var cmd tea.Cmd
		p.shieldInput, cmd = p.shieldInput.Update(msg)
		return p, cmd

	case fieldCMakeArgs:
		if keyStr == "enter" {
			return p, p.startBuild()
		}
		var cmd tea.Cmd
		p.cmakeInput, cmd = p.cmakeInput.Update(msg)
		return p, cmd
	}

	return p, nil
}

func (p *BuildPage) advanceField(dir int) {
	p.blurCurrent()
	p.focusedField = formField((int(p.focusedField) + int(fieldCount) + dir) % int(fieldCount))
	if p.focusedField != fieldBoard {
		p.boardListOpen = false
	}
	p.focusCurrent()
}

func (p *BuildPage) blurAll() {
	p.projectInput.Blur()
	p.boardInput.Blur()
	p.shieldInput.Blur()
	p.cmakeInput.Blur()
	p.boardListOpen = false
}

func (p *BuildPage) blurCurrent() {
	switch p.focusedField {
	case fieldProject:
		p.projectInput.Blur()
	case fieldBoard:
		p.boardInput.Blur()
	case fieldShield:
		p.shieldInput.Blur()
	case fieldCMakeArgs:
		p.cmakeInput.Blur()
	}
}

func (p *BuildPage) focusCurrent() {
	switch p.focusedField {
	case fieldProject:
		p.projectInput.Focus()
	case fieldBoard:
		p.boardInput.Focus()
	case fieldShield:
		p.shieldInput.Focus()
	case fieldCMakeArgs:
		p.cmakeInput.Focus()
	}
}

func (p *BuildPage) View() string {
	leftWidth := p.width * 40 / 100
	if leftWidth < minLeftWidth {
		leftWidth = minLeftWidth
	}
	if leftWidth > maxLeftWidth {
		leftWidth = maxLeftWidth
	}
	rightWidth := p.width - leftWidth - 2 // gap

	left := p.viewForm(leftWidth)
	right := p.viewOutput(rightWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (p *BuildPage) viewForm(width int) string {
	var b strings.Builder
	b.WriteString(ui.Title("Build"))
	b.WriteString("\n")

	if p.loading {
		b.WriteString("Loading boards...")
		return b.String()
	}

	if p.message != "" {
		b.WriteString(p.message + "\n\n")
	}

	inputWidth := width - labelWidth - 4 // padding
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Temporarily set widths for rendering
	p.projectInput.Width = inputWidth
	p.boardInput.Width = inputWidth
	p.shieldInput.Width = inputWidth
	p.cmakeInput.Width = inputWidth

	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	renderLabel := func(name string, field formField) string {
		padded := fmt.Sprintf("%-9s", name)
		if p.focusedField == field {
			return focusedLabel.Render(padded)
		}
		return normalLabel.Render(padded)
	}

	// Project
	b.WriteString(renderLabel("Project", fieldProject) + " " + p.projectInput.View() + "\n")

	// Board
	b.WriteString(renderLabel("Board", fieldBoard) + " " + p.boardInput.View() + "\n")

	// Board dropdown
	if p.focusedField == fieldBoard && len(p.filteredBoards) > 0 {
		dropdown := p.renderBoardDropdown(inputWidth)
		b.WriteString(dropdown)
	}

	// Shield
	b.WriteString(renderLabel("Shield", fieldShield) + " " + p.shieldInput.View() + "\n")

	// Pristine
	check := "[ ]"
	if p.pristine {
		check = "[x]"
	}
	if p.focusedField == fieldPristine {
		check = focusedLabel.Render(check)
	}
	b.WriteString(renderLabel("Pristine", fieldPristine) + " " + check + "\n")

	// CMake args
	b.WriteString(renderLabel("CMake", fieldCMakeArgs) + " " + p.cmakeInput.View() + "\n")

	b.WriteString("\n")
	b.WriteString(ui.DimStyle.Render("ctrl+b: build  tab: next field  esc: unfocus"))

	return b.String()
}

func (p *BuildPage) renderBoardDropdown(width int) string {
	var b strings.Builder
	padding := strings.Repeat(" ", labelWidth+1)

	count := len(p.filteredBoards)
	visible := count
	if visible > maxDropdownItems {
		visible = maxDropdownItems
	}

	// Scroll window around cursor
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

func (p *BuildPage) viewOutput(width int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(p.height - 2).
		PaddingLeft(1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeft(true).
		BorderForeground(ui.Surface)

	if p.output.Len() == 0 {
		content := ui.DimStyle.Render("Build output will appear here...")
		return style.Render(content)
	}

	return style.Render(p.viewport.View())
}

func (p *BuildPage) Name() string { return "Build" }

func (p *BuildPage) ShortHelp() []key.Binding {
	if p.state == buildStateRunning {
		return []key.Binding{
			key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "build")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "unfocus")),
	}
}

func (p *BuildPage) SetSize(w, h int) {
	p.width = w
	p.height = h

	rightWidth := w - maxLeftWidth - 2
	if rightWidth < 20 {
		rightWidth = 20
	}
	p.viewport.Width = rightWidth - 4
	p.viewport.Height = h - 4
}

func (p *BuildPage) filterBoards() {
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

func (p *BuildPage) projectValue() string {
	v := p.projectInput.Value()
	if v == "" {
		return "."
	}
	return v
}

func (p *BuildPage) startBuild() tea.Cmd {
	board := p.boardInput.Value()
	if board == "" {
		p.message = "Board is required"
		return nil
	}

	p.selectedBoard = board
	p.state = buildStateRunning
	p.output.Reset()
	p.buildStart = time.Now()
	p.message = ""

	project := p.projectValue()
	// Resolve relative project paths against the original CWD, not the workspace root
	if !filepath.IsAbs(project) {
		project = filepath.Join(p.cwd, project)
	}

	args := []string{"build", "-b", board}
	if p.pristine {
		args = append(args, "-p", "always")
	}
	if shield := p.shieldInput.Value(); shield != "" {
		args = append(args, "--shield", shield)
	}
	if cmake := p.cmakeInput.Value(); cmake != "" {
		args = append(args, "--")
		args = append(args, strings.Fields(cmake)...)
	}
	args = append(args, project)

	label := fmt.Sprintf("Building for %s", board)
	if p.pristine {
		label = fmt.Sprintf("Building (pristine) for %s", board)
	}
	p.output.WriteString(label + "...\n\n")
	p.viewport.SetContent(p.output.String())

	return west.RunStreaming("west", args...)
}
