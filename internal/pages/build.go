package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type buildState int

const (
	buildStateBoards buildState = iota
	buildStateRunning
	buildStateDone
)

type BuildPage struct {
	boards        []west.Board
	filtered      []west.Board
	cursor        int
	state         buildState
	search        textinput.Model
	searching     bool
	output        strings.Builder
	viewport      viewport.Model
	store         *store.Store
	selectedBoard string
	buildStart    time.Time
	width, height int
	message       string
	loading       bool
}

func NewBuildPage(s *store.Store) *BuildPage {
	ti := textinput.New()
	ti.Placeholder = "Search boards..."
	ti.CharLimit = 64
	vp := viewport.New(0, 0)

	return &BuildPage{
		search:   ti,
		viewport: vp,
		store:    s,
	}
}

func (p *BuildPage) Init() tea.Cmd {
	p.loading = true
	return west.ListBoards()
}

func (p *BuildPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case west.BoardsLoadedMsg:
		p.loading = false
		if msg.Err != nil {
			p.message = fmt.Sprintf("Error loading boards: %v", msg.Err)
			return p, nil
		}
		p.boards = msg.Boards
		p.filterBoards()
		return p, nil

	case west.CommandResultMsg:
		p.state = buildStateDone
		p.output.WriteString(msg.Output)
		success := msg.ExitCode == 0
		status := "success"
		if !success {
			status = fmt.Sprintf("failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\nBuild %s in %s\n", status, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		// Record build
		if p.store != nil {
			p.store.AddBuild(store.BuildRecord{
				Board:     p.selectedBoard,
				App:       ".",
				Timestamp: p.buildStart,
				Success:   success,
				Duration:  msg.Duration.String(),
			})
		}
		return p, nil

	case tea.KeyMsg:
		if p.searching {
			switch msg.String() {
			case "enter", "esc":
				p.searching = false
				p.search.Blur()
				return p, nil
			}
			var cmd tea.Cmd
			p.search, cmd = p.search.Update(msg)
			p.filterBoards()
			return p, cmd
		}

		if p.state == buildStateRunning {
			var cmd tea.Cmd
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "/":
			p.searching = true
			p.search.Focus()
			return p, p.search.Focus()
		case "j", "down":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "b", "enter":
			return p, p.startBuild(false)
		case "p":
			return p, p.startBuild(true)
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
			p.state = buildStateBoards
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *BuildPage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Build"))
	b.WriteString("\n")

	if p.loading {
		b.WriteString("Loading boards...")
		return b.String()
	}

	if p.message != "" {
		b.WriteString(p.message + "\n\n")
	}

	if p.searching {
		b.WriteString(p.search.View())
		b.WriteString("\n\n")
	}

	// Board list (left side in compact view)
	listHeight := p.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	if p.state == buildStateBoards || p.state == buildStateDone {
		if len(p.filtered) == 0 {
			b.WriteString(ui.DimStyle.Render("No boards found"))
			b.WriteString("\n")
		} else {
			// Show boards around cursor
			start := p.cursor - listHeight/2
			if start < 0 {
				start = 0
			}
			end := start + listHeight
			if end > len(p.filtered) {
				end = len(p.filtered)
				start = end - listHeight
				if start < 0 {
					start = 0
				}
			}

			for i := start; i < end; i++ {
				cursor := "  "
				if i == p.cursor {
					cursor = ui.BoldStyle.Render("> ")
				}
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, p.filtered[i].Name))
			}

			b.WriteString(fmt.Sprintf("\n  %d/%d boards", p.cursor+1, len(p.filtered)))
			if p.search.Value() != "" {
				b.WriteString(fmt.Sprintf(" (filter: %s)", p.search.Value()))
			}
			b.WriteString("\n")
		}
	}

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
	}

	return b.String()
}

func (p *BuildPage) Name() string { return "Build" }

func (p *BuildPage) ShortHelp() []key.Binding {
	if p.searching {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "done")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "build")),
		key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pristine")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *BuildPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h / 3
	if vpHeight < 5 {
		vpHeight = 5
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}

func (p *BuildPage) filterBoards() {
	query := strings.ToLower(p.search.Value())
	if query == "" {
		p.filtered = p.boards
	} else {
		p.filtered = nil
		for _, b := range p.boards {
			if strings.Contains(strings.ToLower(b.Name), query) {
				p.filtered = append(p.filtered, b)
			}
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *BuildPage) startBuild(pristine bool) tea.Cmd {
	if len(p.filtered) == 0 {
		return nil
	}

	board := p.filtered[p.cursor]
	p.selectedBoard = board.Name
	p.state = buildStateRunning
	p.output.Reset()
	p.buildStart = time.Now()

	args := []string{"build", "-b", board.Name, "."}
	if pristine {
		args = []string{"build", "-p", "always", "-b", board.Name, "."}
		p.output.WriteString(fmt.Sprintf("Building (pristine) for %s...\n\n", board.Name))
	} else {
		p.output.WriteString(fmt.Sprintf("Building for %s...\n\n", board.Name))
	}
	p.viewport.SetContent(p.output.String())

	return west.RunStreaming("west", args...)
}
