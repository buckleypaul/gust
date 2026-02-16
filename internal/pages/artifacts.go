package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
)

type artifactTab int

const (
	tabBuilds artifactTab = iota
	tabFlashes
	tabTests
	tabSerialLogs
)

var tabNames = []string{"Builds", "Flashes", "Tests", "Serial Logs"}

type ArtifactsPage struct {
	store         *store.Store
	activeTab     artifactTab
	width, height int
}

func NewArtifactsPage(s *store.Store) *ArtifactsPage {
	return &ArtifactsPage{store: s}
}

func (p *ArtifactsPage) Init() tea.Cmd { return nil }

func (p *ArtifactsPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l", "right":
			p.activeTab = (p.activeTab + 1) % artifactTab(len(tabNames))
		case "h", "left":
			p.activeTab = (p.activeTab - 1 + artifactTab(len(tabNames))) % artifactTab(len(tabNames))
		}
	}
	return p, nil
}

func (p *ArtifactsPage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Artifacts"))
	b.WriteString("\n")

	// Tab bar
	for i, name := range tabNames {
		if artifactTab(i) == p.activeTab {
			b.WriteString(ui.BoldStyle.Render(" ["+name+"] "))
		} else {
			b.WriteString(ui.DimStyle.Render("  "+name+"  "))
		}
	}
	b.WriteString("\n\n")

	// Content per tab
	switch p.activeTab {
	case tabBuilds:
		p.renderBuilds(&b)
	case tabFlashes:
		p.renderFlashes(&b)
	case tabTests:
		p.renderTests(&b)
	case tabSerialLogs:
		p.renderSerialLogs(&b)
	}

	return b.String()
}

func (p *ArtifactsPage) renderBuilds(b *strings.Builder) {
	builds, err := p.store.Builds()
	if err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", err))
		return
	}
	if len(builds) == 0 {
		b.WriteString(ui.DimStyle.Render("No build records yet."))
		return
	}
	for i := len(builds) - 1; i >= 0; i-- {
		r := builds[i]
		status := ui.SuccessBadge("OK")
		if !r.Success {
			status = ui.ErrorBadge("FAIL")
		}
		b.WriteString(fmt.Sprintf("  %s  %-30s  %s  %s\n",
			r.Timestamp.Format("Jan 02 15:04"),
			r.Board, r.Duration, status))
	}
}

func (p *ArtifactsPage) renderFlashes(b *strings.Builder) {
	flashes, err := p.store.Flashes()
	if err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", err))
		return
	}
	if len(flashes) == 0 {
		b.WriteString(ui.DimStyle.Render("No flash records yet."))
		return
	}
	for i := len(flashes) - 1; i >= 0; i-- {
		r := flashes[i]
		status := ui.SuccessBadge("OK")
		if !r.Success {
			status = ui.ErrorBadge("FAIL")
		}
		b.WriteString(fmt.Sprintf("  %s  %-30s  %s  %s\n",
			r.Timestamp.Format("Jan 02 15:04"),
			r.Board, r.Duration, status))
	}
}

func (p *ArtifactsPage) renderTests(b *strings.Builder) {
	tests, err := p.store.Tests()
	if err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", err))
		return
	}
	if len(tests) == 0 {
		b.WriteString(ui.DimStyle.Render("No test records yet."))
		return
	}
	for i := len(tests) - 1; i >= 0; i-- {
		r := tests[i]
		status := ui.SuccessBadge("PASS")
		if !r.Success {
			status = ui.ErrorBadge("FAIL")
		}
		b.WriteString(fmt.Sprintf("  %s  %-30s  %s  %s\n",
			r.Timestamp.Format("Jan 02 15:04"),
			r.Board, r.Duration, status))
	}
}

func (p *ArtifactsPage) renderSerialLogs(b *strings.Builder) {
	logs, err := p.store.SerialLogs()
	if err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n", err))
		return
	}
	if len(logs) == 0 {
		b.WriteString(ui.DimStyle.Render("No serial logs yet."))
		return
	}
	for i := len(logs) - 1; i >= 0; i-- {
		r := logs[i]
		b.WriteString(fmt.Sprintf("  %s  %-20s  %d baud  %s\n",
			r.Timestamp.Format("Jan 02 15:04"),
			r.Port, r.BaudRate, r.LogFile))
	}
}

func (p *ArtifactsPage) Name() string { return "Artifacts" }

func (p *ArtifactsPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("h/l"), key.WithHelp("h/l", "switch tab")),
	}
}

func (p *ArtifactsPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}
