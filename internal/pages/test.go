package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type TestPage struct {
	width, height int
}

func NewTestPage() *TestPage {
	return &TestPage{}
}

func (p *TestPage) Init() tea.Cmd { return nil }

func (p *TestPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *TestPage) View() string {
	return ui.Title("Test") + "\n\nTest runner will appear here."
}

func (p *TestPage) Name() string { return "Test" }

func (p *TestPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "run tests")),
	}
}

func (p *TestPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
