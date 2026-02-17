package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type westCommand struct {
	name string
	desc string
	cmd  func() tea.Cmd
}

type WestPage struct {
	commands   []westCommand
	cursor     int
	running    bool
	output     strings.Builder
	viewport   viewport.Model
	width, height int
}

func NewWestPage() *WestPage {
	vp := viewport.New(0, 0)
	return &WestPage{
		commands: []westCommand{
			{"status", "Show workspace status", west.Status},
			{"list", "List workspace projects", west.List},
			{"diff", "Show workspace diffs", west.Diff},
			{"update", "Update workspace", west.Update},
		},
		viewport: vp,
	}
}

func (p *WestPage) Init() tea.Cmd { return nil }

func (p *WestPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.running {
			// While running, only allow viewport scrolling
			var cmd tea.Cmd
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "down":
			if p.cursor < len(p.commands)-1 {
				p.cursor++
			}
		case "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "enter":
			p.running = true
			p.output.Reset()
			p.output.WriteString(fmt.Sprintf("Running west %s...\n\n", p.commands[p.cursor].name))
			p.viewport.SetContent(p.output.String())
			return p, p.commands[p.cursor].cmd()
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
		}

	case west.CommandOutputMsg:
		p.output.WriteString(msg.Line + "\n")
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

	case west.CommandCompletedMsg:
		p.running = false
		if msg.Err != nil {
			p.output.WriteString(fmt.Sprintf("\nError: %v\n", msg.Err))
		}
		p.output.WriteString(fmt.Sprintf("\nCompleted in %s (exit code: %d)\n", msg.Duration, msg.ExitCode))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

	// Handle the bundled result from RunStreaming
	case west.CommandResultMsg:
		// Only handle command results if we're actually running a west command
		if !p.running {
			return p, nil
		}

		p.running = false
		p.output.WriteString(msg.Output)
		status := "success"
		if msg.ExitCode != 0 {
			status = fmt.Sprintf("failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\nCompleted in %s — %s\n", msg.Duration, status))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *WestPage) View() string {
	var b strings.Builder

	// Command panel
	var cmdB strings.Builder
	for i, c := range p.commands {
		cursor := "  "
		if i == p.cursor {
			cursor = ui.BoldStyle.Render("> ")
		}
		cmdB.WriteString(fmt.Sprintf("%s%-10s %s\n", cursor, c.name, ui.DimStyle.Render(c.desc)))
	}
	b.WriteString(ui.Panel("Command", cmdB.String(), p.width, 0, false))

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(ui.Panel("Output", p.viewport.View(), p.width, 0, false))
	}

	return b.String()
}

func (p *WestPage) Name() string { return "West" }

func (p *WestPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *WestPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	// Viewport gets remaining space after command list and title
	vpHeight := h - 10
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}

