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

type WorkspacePage struct {
	workspace     *west.Workspace
	updating      bool
	output        strings.Builder
	viewport      viewport.Model
	width, height int
	message       string
}

func NewWorkspacePage(ws *west.Workspace) *WorkspacePage {
	vp := viewport.New(0, 0)
	return &WorkspacePage{
		workspace: ws,
		viewport:  vp,
	}
}

func (p *WorkspacePage) Init() tea.Cmd { return nil }

func (p *WorkspacePage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.updating {
			var cmd tea.Cmd
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "u":
			p.updating = true
			p.output.Reset()
			p.output.WriteString("Running west update...\n\n")
			p.viewport.SetContent(p.output.String())
			return p, west.Update()
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
			p.message = ""
		}

	case west.CommandResultMsg:
		p.updating = false
		p.output.WriteString(msg.Output)
		if msg.ExitCode == 0 {
			p.message = "Update completed successfully"
		} else {
			p.message = fmt.Sprintf("Update failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\n%s in %s\n", p.message, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()
		return p, nil
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *WorkspacePage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Workspace"))
	b.WriteString("\n")

	ws := p.workspace
	if ws == nil {
		b.WriteString("  No workspace detected.\n")
		return b.String()
	}

	if !ws.Initialized {
		b.WriteString("  " + ui.ErrorBadge("Not Initialized") + "\n\n")
		b.WriteString("  Workspace found but not initialized.\n")
		b.WriteString("  Run `west init` and `west update` to set up.\n")
		return b.String()
	}

	b.WriteString("  " + ui.SuccessBadge("Initialized") + "\n\n")
	b.WriteString(fmt.Sprintf("  Root:     %s\n", ws.Root))
	b.WriteString(fmt.Sprintf("  Manifest: %s\n", ws.ManifestPath))

	if p.message != "" {
		b.WriteString("\n  " + p.message + "\n")
	}

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
	}

	return b.String()
}

func (p *WorkspacePage) Name() string { return "Workspace" }

func (p *WorkspacePage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "west update")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *WorkspacePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h - 10
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}
