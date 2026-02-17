package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary    = lipgloss.Color("38")  // Teal #00afd7
	Secondary  = lipgloss.Color("117") // Sky blue
	Accent     = lipgloss.Color("214") // Amber
	Success    = lipgloss.Color("78")  // Green
	Warning    = lipgloss.Color("214") // Orange
	Error      = lipgloss.Color("196") // Red
	Subtle     = lipgloss.Color("240") // Gray
	Surface    = lipgloss.Color("235") // Dark gray
	Background = lipgloss.Color("234") // Near-black
	Text       = lipgloss.Color("252") // Light gray
	TextDim    = lipgloss.Color("245") // Dimmer text

	// BorderActive is the teal border color for focused panels (alias of Primary).
	BorderActive = Primary

	// Sidebar styles
	SidebarStyle = lipgloss.NewStyle().
			Width(20).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(0, 1)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(TextDim).
				PaddingLeft(1)

	SidebarActiveStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				PaddingLeft(1)

	// Content area — borders are handled by Panel() in layout.
	ContentStyle = lipgloss.NewStyle().
			Padding(0, 0)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextDim).
			Background(Surface).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(Surface).
				Bold(true)

	// Page title
	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginBottom(1)

	// General
	BoldStyle   = lipgloss.NewStyle().Bold(true)
	DimStyle    = lipgloss.NewStyle().Foreground(TextDim)
	AccentStyle = lipgloss.NewStyle().Foreground(Accent)
)
