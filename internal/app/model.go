package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
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
	picker          *Picker
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
		contentWidth := m.width - sidebarWidth
		contentHeight := m.height - 2 - 1 // status bar + project bar
		for _, p := range m.pages {
			p.SetSize(contentWidth, contentHeight)
		}
		return m, nil

	case west.ProjectsLoadedMsg:
		if msg.Err != nil || m.picker == nil {
			return m, nil
		}
		var items []PickerItem
		for _, p := range msg.Projects {
			items = append(items, PickerItem{
				Label: p.Path,
				Value: p.Path,
				Desc:  p.Source,
			})
		}
		m.picker.SetItems(items)
		return m, nil

	case PickerSelectedMsg:
		m.selectedProject = msg.Value
		m.picker = nil
		// Persist to config
		m.cfg.LastProject = msg.Value
		config.Save(*m.cfg, m.wsRoot, false)
		// Broadcast to all pages
		return m, func() tea.Msg { return ProjectSelectedMsg{Path: msg.Value} }

	case PickerClosedMsg:
		m.picker = nil
		return m, nil

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
		// When picker is open, forward all keys to picker
		if m.picker != nil {
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Update(msg)
			return m, cmd
		}

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

		// Sidebar-only shortcuts
		if m.focus == FocusSidebar {
			if key.Matches(msg, GlobalKeys.ProjectPicker) {
				m.picker = NewPicker("Select Project")
				contentWidth := m.width - sidebarWidth
				contentHeight := m.height - 2 - 1
				m.picker.SetSize(contentWidth, contentHeight)
				return m, west.ListProjects(m.wsRoot, m.manifestPath)
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
	contentHeight := m.height - 2 - 1 // status bar + project bar

	page := m.pages[m.activePage]

	projectBar := renderProjectBar(m.selectedProject, m.selectedBoard, m.width, m.focus == FocusSidebar)
	sidebar := renderSidebar(PageOrder, m.activePage, m.pages, contentHeight, m.focus == FocusSidebar)
	content := ui.ContentStyle.
		Width(contentWidth).
		Height(contentHeight).
		Render(page.View())

	// Overlay picker on content area when open
	if m.picker != nil {
		m.picker.SetSize(contentWidth, contentHeight)
		pickerView := m.picker.View()
		content = lipgloss.Place(
			contentWidth, contentHeight,
			lipgloss.Center, lipgloss.Center,
			pickerView,
		)
	}

	statusBar := renderStatusBar(page.ShortHelp(), m.width, m.focus)

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

