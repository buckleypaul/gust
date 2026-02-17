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
	store           *store.Store
	runner          west.Runner
	running         bool
	output          strings.Builder
	viewport        viewport.Model
	testStart       time.Time
	width, height   int
	message         string
	requestSeq      int
	activeRequestID string
}

func NewTestPage(s *store.Store, runners ...west.Runner) *TestPage {
	vp := viewport.New(0, 0)
	runner := west.RealRunner()
	if len(runners) > 0 && runners[0] != nil {
		runner = runners[0]
	}
	return &TestPage{
		store:    s,
		runner:   runner,
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
			requestID := p.nextRequestID()
			p.activeRequestID = requestID
			p.output.Reset()
			p.testStart = time.Now()
			p.output.WriteString("Running tests...\n\n")
			p.viewport.SetContent(p.output.String())
			return p, west.WithRequestID(requestID, p.runner.Run("west", "build", "-t", "run"))
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
		if msg.RequestID != p.activeRequestID {
			return p, nil
		}

		p.running = false
		p.activeRequestID = ""
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
			if err := p.store.AddTest(store.TestRecord{
				Timestamp: p.testStart,
				Success:   success,
				Duration:  msg.Duration.String(),
			}); err != nil {
				p.message = fmt.Sprintf("Tests completed, but history save failed: %v", err)
			}
		}
		return p, nil
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *TestPage) View() string {
	var b strings.Builder

	// Configuration/status panel
	var cfgB strings.Builder
	if p.message != "" {
		cfgB.WriteString("  " + p.message + "\n")
	}
	if !p.running && p.output.Len() == 0 {
		cfgB.WriteString(ui.DimStyle.Render("  Press t or Enter to run tests."))
		cfgB.WriteString("\n")
	}
	b.WriteString(ui.Panel("Configuration", cfgB.String(), p.width, 0, false))

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(ui.Panel("Output", p.viewport.View(), p.width, 0, false))
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

func (p *TestPage) nextRequestID() string {
	p.requestSeq++
	return fmt.Sprintf("test-%d", p.requestSeq)
}
