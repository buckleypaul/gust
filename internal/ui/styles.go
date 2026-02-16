package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary    = lipgloss.Color("63")  // Purple/blue
	Secondary  = lipgloss.Color("86")  // Cyan
	Accent     = lipgloss.Color("205") // Pink
	Success    = lipgloss.Color("78")  // Green
	Warning    = lipgloss.Color("214") // Orange
	Error      = lipgloss.Color("196") // Red
	Subtle     = lipgloss.Color("241") // Gray
	Surface    = lipgloss.Color("236") // Dark gray
	Background = lipgloss.Color("235") // Darker gray
	Text       = lipgloss.Color("252") // Light gray
	TextDim    = lipgloss.Color("245") // Dimmer text

	// Sidebar styles
	SidebarStyle = lipgloss.NewStyle().
			Width(20).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderRight(true).
			BorderTop(false).
			BorderBottom(false).
			BorderLeft(false).
			BorderForeground(Surface).
			Padding(1, 1)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(TextDim).
				PaddingLeft(1)

	SidebarActiveStyle = lipgloss.NewStyle().
				Foreground(Primary).
				Bold(true).
				PaddingLeft(1)

	// Content area
	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2)

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
