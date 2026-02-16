package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type BuildPage struct {
	width, height int
}

func NewBuildPage() *BuildPage {
	return &BuildPage{}
}

func (p *BuildPage) Init() tea.Cmd { return nil }

func (p *BuildPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *BuildPage) View() string {
	return ui.Title("Build") + "\n\nBoard selection and build output will appear here."
}

func (p *BuildPage) Name() string { return "Build" }

func (p *BuildPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "build")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pristine")),
	}
}

func (p *BuildPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
