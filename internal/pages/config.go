package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type ConfigPage struct {
	width, height int
}

func NewConfigPage() *ConfigPage {
	return &ConfigPage{}
}

func (p *ConfigPage) Init() tea.Cmd { return nil }

func (p *ConfigPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *ConfigPage) View() string {
	return ui.Title("Config") + "\n\nKconfig browser will appear here."
}

func (p *ConfigPage) Name() string { return "Config" }

func (p *ConfigPage) ShortHelp() []key.Binding { return nil }

func (p *ConfigPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
