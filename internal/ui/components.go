package ui

import "github.com/charmbracelet/lipgloss"

// Title renders a styled page title.
func Title(text string) string {
	return TitleStyle.Render(text)
}

// StatusKey renders a key hint for the status bar.
func StatusKey(k, desc string) string {
	return StatusBarKeyStyle.Render(k) + StatusBarStyle.Render(":"+desc)
}

// Badge renders a small colored badge.
func Badge(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(color).
		Padding(0, 1).
		Render(text)
}

// SuccessBadge renders a green badge.
func SuccessBadge(text string) string {
	return Badge(text, Success)
}

// ErrorBadge renders a red badge.
func ErrorBadge(text string) string {
	return Badge(text, Error)
}
