package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/ui"
)

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
)

type Model struct {
	pages           map[PageID]Page
	activePage      PageID
	focus           FocusArea
	width           int
	height          int
	showHelp        bool
	selectedProject string
	selectedBoard   string
	selectedShield  string
	cfg             *config.Config
	wsRoot          string
	manifestPath    string
}

func New(pages map[PageID]Page, cfg *config.Config, wsRoot string, manifestPath string) Model {
	return Model{
		pages:           pages,
		cfg:             cfg,
		wsRoot:          wsRoot,
		manifestPath:    manifestPath,
		selectedProject: cfg.LastProject,
		selectedBoard:   cfg.DefaultBoard,
		selectedShield:  cfg.LastShield,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.pages {
		if cmd := p.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentWidth := m.width - sidebarWidth   // outer width of content panel
		innerWidth := contentWidth - 4            // -2 border -2 padding
		contentHeight := m.height - 2 - 1 - 2    // status bar + project bar + top/bottom panel border
		if innerWidth < 0 {
			innerWidth = 0
		}
		if contentHeight < 0 {
			contentHeight = 0
		}
		for _, p := range m.pages {
			p.SetSize(innerWidth, contentHeight)
		}
		return m, nil

	case ProjectSelectedMsg:
		m.selectedProject = msg.Path
		// Forward to all pages
		var cmds []tea.Cmd
		for id, page := range m.pages {
			newPage, cmd := page.Update(msg)
			m.pages[id] = newPage
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case BoardSelectedMsg:
		m.selectedBoard = msg.Board
		// Broadcast to all pages
		var cmds []tea.Cmd
		for id, page := range m.pages {
			newPage, cmd := page.Update(msg)
			m.pages[id] = newPage
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case ShieldSelectedMsg:
		m.selectedShield = msg.Shield
		// Broadcast to all pages
		var cmds []tea.Cmd
		for id, page := range m.pages {
			newPage, cmd := page.Update(msg)
			m.pages[id] = newPage
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// When a page has an active text input, forward all keys
		// directly to the page — only ctrl+c still quits.
		if m.focus == FocusContent {
			if ic, ok := m.pages[m.activePage].(InputCapturer); ok && ic.InputCaptured() {
				if msg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				page := m.pages[m.activePage]
				newPage, cmd := page.Update(msg)
				m.pages[m.activePage] = newPage
				return m, cmd
			}
		}

		// Global key handling
		switch {
		case key.Matches(msg, GlobalKeys.Quit):
			return m, tea.Quit
		case key.Matches(msg, GlobalKeys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, GlobalKeys.ToggleFocus):
			if m.focus == FocusSidebar {
				m.focus = FocusContent
				return m, nil
			}
			// When content focused, fall through to page handler
		}

		// Handle arrow keys based on focus
		if m.focus == FocusSidebar {
			switch msg.String() {
			case "up":
				m.prevPage()
				return m, nil
			case "down":
				m.nextPage()
				return m, nil
			case "enter", "right":
				m.focus = FocusContent
				return m, nil
			}
		} else if m.focus == FocusContent {
			if msg.String() == "left" {
				m.focus = FocusSidebar
				return m, nil
			}
		}
	}

	// Key messages: only forward to active page when content is focused
	if _, isKey := msg.(tea.KeyMsg); isKey {
		if m.focus != FocusContent {
			return m, nil
		}
		page := m.pages[m.activePage]
		newPage, cmd := page.Update(msg)
		m.pages[m.activePage] = newPage
		return m, cmd
	}

	// Non-key messages (command results, etc.): forward to all pages
	// so responses reach the page that initiated the command
	var cmds []tea.Cmd
	for id, page := range m.pages {
		newPage, cmd := page.Update(msg)
		m.pages[id] = newPage
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	contentWidth := m.width - sidebarWidth   // outer width of content panel
	contentHeight := m.height - 2 - 1        // status bar + project bar

	page := m.pages[m.activePage]

	projectBar := renderProjectBar(m.selectedProject, m.selectedBoard, m.width)
	sidebar := renderSidebar(PageOrder, m.activePage, m.pages, contentHeight, m.focus == FocusSidebar)
	content := ui.Panel(page.Name(), page.View(), contentWidth, contentHeight, m.focus == FocusContent)

	statusBar := renderStatusBar(page.ShortHelp(), m.width, m.focus, m.wsRoot)

	return renderLayout(projectBar, sidebar, content, statusBar)
}

func (m *Model) nextPage() {
	for i, id := range PageOrder {
		if id == m.activePage {
			m.activePage = PageOrder[(i+1)%len(PageOrder)]
			return
		}
	}
}

func (m *Model) prevPage() {
	for i, id := range PageOrder {
		if id == m.activePage {
			m.activePage = PageOrder[(i-1+len(PageOrder))%len(PageOrder)]
			return
		}
	}
}
