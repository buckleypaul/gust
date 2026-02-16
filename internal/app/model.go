package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/ui"
)

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
)

type Model struct {
	pages      map[PageID]Page
	activePage PageID
	focus      FocusArea
	width      int
	height     int
	showHelp   bool
}

func New(pages map[PageID]Page) Model {
	return Model{
		pages:      pages,
		activePage: WorkspacePage,
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
		contentWidth := m.width - sidebarWidth
		contentHeight := m.height - 2 // status bar
		for _, p := range m.pages {
			p.SetSize(contentWidth, contentHeight)
		}
		return m, nil

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

		// Number keys and other shortcuts only when sidebar is focused
		if m.focus == FocusSidebar {
			switch {
			case key.Matches(msg, GlobalKeys.Page1):
				m.setPage(0)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page2):
				m.setPage(1)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page3):
				m.setPage(2)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page4):
				m.setPage(3)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page5):
				m.setPage(4)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page6):
				m.setPage(5)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page7):
				m.setPage(6)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page8):
				m.setPage(7)
				return m, nil
			case key.Matches(msg, GlobalKeys.Page9):
				m.setPage(8)
				return m, nil
			}
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

	contentWidth := m.width - sidebarWidth
	contentHeight := m.height - 2

	page := m.pages[m.activePage]

	sidebar := renderSidebar(PageOrder, m.activePage, m.pages, contentHeight, m.focus == FocusSidebar)
	content := ui.ContentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(page.View())
	statusBar := renderStatusBar(page.ShortHelp(), m.width, m.focus)

	return renderLayout(sidebar, content, statusBar)
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

func (m *Model) setPage(idx int) {
	if idx >= 0 && idx < len(PageOrder) {
		m.activePage = PageOrder[idx]
	}
}
