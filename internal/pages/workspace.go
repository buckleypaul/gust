package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type WorkspacePage struct {
	workspace    *west.Workspace
	width, height int
}

func NewWorkspacePage(ws *west.Workspace) *WorkspacePage {
	return &WorkspacePage{workspace: ws}
}

func (p *WorkspacePage) Init() tea.Cmd { return nil }

func (p *WorkspacePage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *WorkspacePage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Workspace"))
	b.WriteString("\n")

	ws := p.workspace
	if ws == nil {
		b.WriteString("No workspace detected.\n")
		return b.String()
	}

	if !ws.Initialized {
		b.WriteString(ui.ErrorBadge("Not Initialized"))
		b.WriteString("\n\n")
		b.WriteString("Workspace found but not initialized.\n")
		b.WriteString("Run `west init` and `west update` to set up.\n")
		return b.String()
	}

	b.WriteString(ui.SuccessBadge("Initialized"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Root:     %s\n", ws.Root))
	b.WriteString(fmt.Sprintf("  Manifest: %s\n", ws.ManifestPath))

	return b.String()
}

func (p *WorkspacePage) Name() string { return "Workspace" }

func (p *WorkspacePage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "west update")),
	}
}

func (p *WorkspacePage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
