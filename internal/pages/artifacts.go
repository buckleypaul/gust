package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type ArtifactsPage struct {
	width, height int
}

func NewArtifactsPage() *ArtifactsPage {
	return &ArtifactsPage{}
}

func (p *ArtifactsPage) Init() tea.Cmd { return nil }

func (p *ArtifactsPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *ArtifactsPage) View() string {
	return ui.Title("Artifacts") + "\n\nBuild artifacts and logs will appear here."
}

func (p *ArtifactsPage) Name() string { return "Artifacts" }

func (p *ArtifactsPage) ShortHelp() []key.Binding { return nil }

func (p *ArtifactsPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
