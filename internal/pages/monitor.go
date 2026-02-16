package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type MonitorPage struct {
	width, height int
}

func NewMonitorPage() *MonitorPage {
	return &MonitorPage{}
}

func (p *MonitorPage) Init() tea.Cmd { return nil }

func (p *MonitorPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *MonitorPage) View() string {
	return ui.Title("Monitor") + "\n\nSerial monitor will appear here."
}

func (p *MonitorPage) Name() string { return "Monitor" }

func (p *MonitorPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "disconnect")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "scroll")),
	}
}

func (p *MonitorPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
