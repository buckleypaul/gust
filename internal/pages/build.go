package pages

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"

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
	case app.ProjectSelectedMsg:
		p.projectInput.SetValue(msg.Path)
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

	case west.CommandResultMsg:
		// Only handle command results if we're actually running a build
		if p.state != buildStateRunning {
			return p, nil
		}

		p.state = buildStateDone
		p.output.WriteString(msg.Output)
		success := msg.ExitCode == 0
		status := "success"
		if !success {
			status = fmt.Sprintf("failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\nBuild %s in %s\n", status, msg.Duration))
		p.updateViewportContent()
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
	case "y":
		if !p.InputCaptured() && p.output.Len() > 0 {
			p.copyToClipboard()
			return p, nil
		}
	case "esc":
		if p.state == buildStateDone {
			p.state = buildStateIdle
			p.output.Reset()
			p.updateViewportContent()
			return p, nil
		}
		p.blurAll()
		return p, nil
	}

	// Field-specific handling
	switch p.focusedField {
	case fieldBoard:
		if keyStr == "down" {
			if len(p.filteredBoards) > 0 && !p.boardListOpen {
				// Open dropdown on first down press
				p.boardListOpen = true
				p.boardCursor = 0
				return p, nil
			} else if !p.boardListOpen {
				// No dropdown, move to next field
				p.advanceField(1)
				return p, nil
			}
		}
		if keyStr == "up" && !p.boardListOpen {
			p.advanceField(-1)
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
		switch keyStr {
		case "enter", " ":
			p.pristine = !p.pristine
			return p, nil
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
		}
		return p, nil

	case fieldProject:
		switch keyStr {
		case "enter":
			return p, p.startBuild()
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
		}
		var cmd tea.Cmd
		p.projectInput, cmd = p.projectInput.Update(msg)
		return p, cmd

	case fieldShield:
		switch keyStr {
		case "enter":
			return p, p.startBuild()
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

	case fieldCMakeArgs:
		switch keyStr {
		case "enter":
			return p, p.startBuild()
		case "up":
			p.advanceField(-1)
			return p, nil
		case "down":
			p.advanceField(1)
			return p, nil
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
	// Split vertically: form on top, output below
	formHeight := 12 // Fixed height for form
	if p.focusedField == fieldBoard && len(p.filteredBoards) > 0 {
		// Add space for board dropdown
		formHeight += maxDropdownItems + 2
	}
	outputHeight := p.height - formHeight - 1 // -1 for separator

	if outputHeight < 5 {
		outputHeight = 5
		formHeight = p.height - outputHeight - 1
	}

	form := p.viewForm(p.width, formHeight)
	output := p.viewOutput(p.width, outputHeight)

	return lipgloss.JoinVertical(lipgloss.Left, form, output)
}

func (p *BuildPage) viewForm(width int, height int) string {
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
	helpText := "ctrl+b: build  tab: next field  esc: unfocus"
	if p.output.Len() > 0 {
		helpText += "  y: copy output"
	}
	b.WriteString(ui.DimStyle.Render(helpText))

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

func (p *BuildPage) viewOutput(width int, height int) string {
	// Account for border (2 chars top+bottom) and padding (1 char left)
	contentWidth := width - 3
	contentHeight := height - 2

	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Update viewport size to match available space
	oldWidth := p.viewport.Width
	p.viewport.Width = contentWidth
	p.viewport.Height = contentHeight

	// Re-wrap content if width changed
	if oldWidth != contentWidth && p.output.Len() > 0 {
		p.updateViewportContent()
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderForeground(ui.Surface).
		PaddingLeft(1).
		PaddingTop(0)

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
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
		key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "build")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "unfocus")),
	}
	if p.output.Len() > 0 {
		bindings = append(bindings, key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy output")))
	}
	return bindings
}

func (p *BuildPage) InputCaptured() bool {
	return p.state == buildStateIdle && p.focusedField != fieldPristine &&
		(p.projectInput.Focused() || p.boardInput.Focused() ||
			p.shieldInput.Focused() || p.cmakeInput.Focused())
}

func (p *BuildPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	// Viewport size will be set dynamically in viewOutput()
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

func (p *BuildPage) updateViewportContent() {
	if p.viewport.Width > 0 {
		// Use hard wrap to handle long paths/commands that don't have spaces
		content := p.output.String()
		wrapped := wrap.String(content, p.viewport.Width)

		// Additional safety: truncate any lines that are still too long (ANSI-aware)
		lines := strings.Split(wrapped, "\n")
		for i, line := range lines {
			// Check printable width (excluding ANSI codes)
			if ansi.PrintableRuneWidth(line) > p.viewport.Width {
				lines[i] = truncate.String(line, uint(p.viewport.Width))
			}
		}
		p.viewport.SetContent(strings.Join(lines, "\n"))
	} else {
		p.viewport.SetContent(p.output.String())
	}
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
	p.updateViewportContent()

	return west.RunStreaming("west", args...)
}

func (p *BuildPage) copyToClipboard() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try wl-copy (Wayland) first, fall back to xclip (X11)
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		p.message = "Clipboard copy not supported on this platform"
		return
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		p.message = fmt.Sprintf("Failed to copy: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		p.message = fmt.Sprintf("Failed to copy: %v", err)
		return
	}

	if _, err := stdin.Write([]byte(p.output.String())); err != nil {
		p.message = fmt.Sprintf("Failed to copy: %v", err)
		stdin.Close()
		cmd.Wait()
		return
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		p.message = fmt.Sprintf("Failed to copy: %v", err)
		return
	}

	p.message = "Build output copied to clipboard"
}
