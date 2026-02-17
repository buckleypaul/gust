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
	fieldPristine formField = iota
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
	cmakeInput textinput.Model
	pristine   bool

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
	selectedProject string
	selectedBoard   string
	selectedShield  string
	buildStart      time.Time
	width, height   int
	message         string
}

func NewBuildPage(s *store.Store, cfg *config.Config, wsRoot string, cwd string) *BuildPage {
	cmake := textinput.New()
	cmake.Placeholder = "e.g. -DOVERLAY_CONFIG=overlay.conf"
	cmake.CharLimit = 512
	cmake.Prompt = ""

	vp := viewport.New(0, 0)

	return &BuildPage{
		cmakeInput:      cmake,
		viewport:        vp,
		store:           s,
		cfg:             cfg,
		wsRoot:          wsRoot,
		cwd:             cwd,
		focusedField:    fieldPristine,
		selectedProject: cfg.LastProject,
		selectedBoard:   cfg.DefaultBoard,
		selectedShield:  cfg.LastShield,
	}
}

func (p *BuildPage) Init() tea.Cmd {
	return nil
}

func (p *BuildPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case app.ProjectSelectedMsg:
		p.selectedProject = msg.Path
		return p, nil

	case app.BoardSelectedMsg:
		p.selectedBoard = msg.Board
		return p, nil

	case app.ShieldSelectedMsg:
		p.selectedShield = msg.Shield
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
				Shield:    p.selectedShield,
				Pristine:  p.pristine,
				CMakeArgs: p.cmakeInput.Value(),
			})
		}

		// Persist board to config on success
		if success {
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
	p.focusCurrent()
}

func (p *BuildPage) blurAll() {
	p.cmakeInput.Blur()
}

func (p *BuildPage) blurCurrent() {
	switch p.focusedField {
	case fieldCMakeArgs:
		p.cmakeInput.Blur()
	}
}

func (p *BuildPage) focusCurrent() {
	switch p.focusedField {
	case fieldCMakeArgs:
		p.cmakeInput.Focus()
	}
}

func (p *BuildPage) View() string {
	// Split vertically: form on top, output below
	formHeight := 10 // Fixed height for form
	outputHeight := p.height - formHeight - 1 // -1 for separator

	if outputHeight < 5 {
		outputHeight = 5
		formHeight = p.height - outputHeight - 1
	}

	form := ui.Panel("Configuration", p.viewForm(p.width, formHeight), p.width, formHeight, false)
	output := p.viewOutput(p.width, outputHeight)

	return lipgloss.JoinVertical(lipgloss.Left, form, output)
}

func (p *BuildPage) viewForm(width int, height int) string {
	var b strings.Builder

	if p.message != "" {
		b.WriteString(p.message + "\n\n")
	}

	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	renderLabel := func(name string, field formField) string {
		padded := fmt.Sprintf("%-9s", name)
		if p.focusedField == field {
			return focusedLabel.Render(padded)
		}
		return normalLabel.Render(padded)
	}

	// Read-only context: project, board, shield
	projectDisplay := p.selectedProject
	if projectDisplay == "" {
		projectDisplay = "."
	}
	boardDisplay := p.selectedBoard
	if boardDisplay == "" {
		boardDisplay = "(none)"
	}
	shieldDisplay := p.selectedShield
	if shieldDisplay == "" {
		shieldDisplay = "(none)"
	}

	infoStyle := lipgloss.NewStyle().Foreground(ui.Text)
	dimStyle := ui.DimStyle

	b.WriteString(normalLabel.Render(fmt.Sprintf("%-9s", "Building")) + " " + infoStyle.Render(projectDisplay) + "\n")
	b.WriteString(normalLabel.Render(fmt.Sprintf("%-9s", "Board")) + " " + infoStyle.Render(boardDisplay) +
		"  " + dimStyle.Render("Shield:") + " " + infoStyle.Render(shieldDisplay) + "\n")
	b.WriteString("\n")

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
	inputWidth := width - labelWidth - 4 // padding
	if inputWidth < 10 {
		inputWidth = 10
	}
	p.cmakeInput.Width = inputWidth
	b.WriteString(renderLabel("CMake", fieldCMakeArgs) + " " + p.cmakeInput.View() + "\n")

	b.WriteString("\n")
	helpText := "ctrl+b: build  tab: next field  esc: unfocus"
	if p.output.Len() > 0 {
		helpText += "  y: copy output"
	}
	b.WriteString(ui.DimStyle.Render(helpText))

	return b.String()
}

func (p *BuildPage) viewOutput(width int, height int) string {
	// Account for panel border (2 top/bottom) and padding (2 left/right) and 2 border sides
	contentWidth := width - 4
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

	var content string
	if p.output.Len() == 0 {
		content = ui.DimStyle.Render("Build output will appear here...")
	} else {
		content = p.viewport.View()
	}

	return ui.Panel("Output", content, width, height, false)
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
	return p.state == buildStateIdle && p.cmakeInput.Focused()
}

func (p *BuildPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	// Viewport size will be set dynamically in viewOutput()
}

func (p *BuildPage) projectValue() string {
	if p.selectedProject == "" {
		return "."
	}
	return p.selectedProject
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
	board := p.selectedBoard
	if board == "" {
		p.message = "Board is required. Select a board on the Project page."
		return nil
	}

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
	if p.selectedShield != "" {
		args = append(args, "--shield", p.selectedShield)
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
