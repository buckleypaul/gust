package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// PageID identifies each page in the application.
type PageID int

const (
	WorkspacePage PageID = iota
	BuildPage
	FlashPage
	MonitorPage
	TestPage
	ArtifactsPage
	WestPage
	ConfigPage
	SettingsPage
)

var PageOrder = []PageID{
	WorkspacePage,
	BuildPage,
	FlashPage,
	MonitorPage,
	TestPage,
	ArtifactsPage,
	WestPage,
	ConfigPage,
	SettingsPage,
}

// Page is the interface every page in the application implements.
type Page interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Page, tea.Cmd)
	View() string
	Name() string
	ShortHelp() []key.Binding
	SetSize(width, height int)
}

// InputCapturer is an optional interface for pages with text inputs.
// When InputCaptured returns true, the app forwards all keys directly
// to the page instead of processing shortcuts like q, ?, left, etc.
type InputCapturer interface {
	InputCaptured() bool
}

// ProjectSelectedMsg is broadcast to all pages when a project is selected.
type ProjectSelectedMsg struct {
	Path string
}
// BoardSelectedMsg is broadcast to all pages when a board is selected.
type BoardSelectedMsg struct {
	Board string
}

// ShieldSelectedMsg is broadcast to all pages when a shield is selected.
type ShieldSelectedMsg struct {
	Shield string
}
