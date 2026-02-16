package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type WestPage struct {
	width, height int
}

func NewWestPage() *WestPage {
	return &WestPage{}
}

func (p *WestPage) Init() tea.Cmd { return nil }

func (p *WestPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *WestPage) View() string {
	return ui.Title("West") + "\n\nWest command runner will appear here."
}

func (p *WestPage) Name() string { return "West" }

func (p *WestPage) ShortHelp() []key.Binding { return nil }

func (p *WestPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
