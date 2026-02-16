package pages

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

type FlashPage struct {
	width, height int
}

func NewFlashPage() *FlashPage {
	return &FlashPage{}
}

func (p *FlashPage) Init() tea.Cmd { return nil }

func (p *FlashPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	return p, nil
}

func (p *FlashPage) View() string {
	return ui.Title("Flash") + "\n\nFlash firmware controls will appear here."
}

func (p *FlashPage) Name() string { return "Flash" }

func (p *FlashPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "flash")),
	}
}

func (p *FlashPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
