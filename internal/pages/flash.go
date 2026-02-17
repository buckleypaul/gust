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

type FlashPage struct {
	store           *store.Store
	runner          west.Runner
	lastBuild       *store.BuildRecord
	selectedProject string
	flashing        bool
	output          strings.Builder
	viewport        viewport.Model
	flashStart      time.Time
	width, height   int
	requestSeq      int
	activeRequestID string
	message         string
}

func NewFlashPage(s *store.Store, runners ...west.Runner) *FlashPage {
	vp := viewport.New(0, 0)
	runner := west.RealRunner()
	if len(runners) > 0 && runners[0] != nil {
		runner = runners[0]
	}
	return &FlashPage{
		store:    s,
		runner:   runner,
		viewport: vp,
	}
}

func (p *FlashPage) Init() tea.Cmd { return nil }

func (p *FlashPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case app.ProjectSelectedMsg:
		p.selectedProject = msg.Path
		return p, nil

	case tea.KeyMsg:
		if p.flashing {
			var cmd tea.Cmd
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "f", "enter":
			p.refreshLastBuild()
			p.flashing = true
			requestID := p.nextRequestID()
			p.activeRequestID = requestID
			p.output.Reset()
			p.flashStart = time.Now()

			board := ""
			if p.lastBuild != nil {
				board = p.lastBuild.Board
			}
			p.output.WriteString(fmt.Sprintf("Flashing %s...\n\n", board))
			p.viewport.SetContent(p.output.String())
			return p, west.WithRequestID(requestID, p.runner.Run("west", "flash"))
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
			p.message = ""
		}

	case west.CommandResultMsg:
		// Only handle command results if we're actually flashing
		if !p.flashing {
			return p, nil
		}
		if msg.RequestID != p.activeRequestID {
			return p, nil
		}

		p.flashing = false
		p.activeRequestID = ""
		p.output.WriteString(msg.Output)
		success := msg.ExitCode == 0
		status := "success"
		if !success {
			status = fmt.Sprintf("failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\nFlash %s in %s\n", status, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		// Record flash
		board := ""
		if p.lastBuild != nil {
			board = p.lastBuild.Board
		}
		if p.store != nil {
			if err := p.store.AddFlash(store.FlashRecord{
				Board:     board,
				Timestamp: p.flashStart,
				Success:   success,
				Duration:  msg.Duration.String(),
			}); err != nil {
				p.message = fmt.Sprintf("Flash finished, but history save failed: %v", err)
			}
		}
		return p, nil
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *FlashPage) View() string {
	p.refreshLastBuild()

	// Status section
	var statusB strings.Builder
	if p.selectedProject != "" {
		statusB.WriteString(fmt.Sprintf("  Project: %s\n", p.selectedProject))
	}
	if p.lastBuild != nil {
		statusB.WriteString(fmt.Sprintf("  Last build: %s (%s)\n",
			p.lastBuild.Board,
			p.lastBuild.Timestamp.Format("2006-01-02 15:04:05")))
		if p.lastBuild.Success {
			statusB.WriteString("  Status: " + ui.SuccessBadge("OK") + "\n")
		} else {
			statusB.WriteString("  Status: " + ui.ErrorBadge("FAILED") + "\n")
		}
	} else {
		statusB.WriteString(ui.DimStyle.Render("  No recent builds found. Build first.") + "\n")
	}
	if p.message != "" {
		statusB.WriteString("  " + p.message + "\n")
	}
	statusPanel := ui.Panel("Status", statusB.String(), p.width, 0, false)

	var b strings.Builder
	b.WriteString(statusPanel)

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(ui.Panel("Output", p.viewport.View(), p.width, 0, false))
	}

	return b.String()
}

func (p *FlashPage) Name() string { return "Flash" }

func (p *FlashPage) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "flash")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *FlashPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h - 8
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}

func (p *FlashPage) refreshLastBuild() {
	if p.store == nil {
		return
	}
	builds, err := p.store.Builds()
	if err != nil || len(builds) == 0 {
		p.lastBuild = nil
		return
	}
	p.lastBuild = &builds[len(builds)-1]
}

func (p *FlashPage) nextRequestID() string {
	p.requestSeq++
	return fmt.Sprintf("flash-%d", p.requestSeq)
}
