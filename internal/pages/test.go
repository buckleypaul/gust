package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type TestPage struct {
	store      *store.Store
	running    bool
	output     strings.Builder
	viewport   viewport.Model
	testStart  time.Time
	width, height int
	message    string
}

func NewTestPage(s *store.Store) *TestPage {
	vp := viewport.New(0, 0)
	return &TestPage{
		store:    s,
		viewport: vp,
	}
}

func (p *TestPage) Init() tea.Cmd { return nil }

func (p *TestPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.running {
			var cmd tea.Cmd
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "t", "enter":
			p.running = true
			p.output.Reset()
			p.testStart = time.Now()
			p.output.WriteString("Running tests...\n\n")
			p.viewport.SetContent(p.output.String())
			return p, west.RunStreaming("west", "build", "-t", "run")
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
			p.message = ""
		}

	case west.CommandResultMsg:
		// Only handle command results if we're actually running tests
		if !p.running {
			return p, nil
		}

		p.running = false
		p.output.WriteString(msg.Output)
		success := msg.ExitCode == 0
		if success {
			p.message = "Tests passed"
		} else {
			p.message = fmt.Sprintf("Tests failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\n%s in %s\n", p.message, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		// Record test result
		if p.store != nil {
			p.store.AddTest(store.TestRecord{
				Timestamp: p.testStart,
				Success:   success,
				Duration:  msg.Duration.String(),
			})
		}
		return p, nil
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *TestPage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Test"))
	b.WriteString("\n")

	if p.message != "" {
		b.WriteString("  " + p.message + "\n\n")
	}

	if !p.running && p.output.Len() == 0 {
		b.WriteString(ui.DimStyle.Render("  Press t or Enter to run tests."))
		b.WriteString("\n")
	}

	if p.output.Len() > 0 {
		b.WriteString(p.viewport.View())
	}

	return b.String()
}

func (p *TestPage) Name() string { return "Test" }

func (p *TestPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "run tests")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *TestPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h - 6
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}
