package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panel renders a rounded-border box with title embedded in the top border.
// width is the total outer width. height=0 means auto-height.
// Border color is Primary when focused, Subtle when not.
func Panel(title, content string, width, height int, focused bool) string {
	borderColor := Subtle
	if focused {
		borderColor = Primary
	}

	colorStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Compute dash count for top border:
	// ╭─ TITLE ─...─╮  total = width
	// 3 (╭─ ) + len(title) + 1 ( ) + dashCount + 1 (╮) = width
	dashCount := width - len(title) - 5
	if dashCount < 0 {
		dashCount = 0
	}

	topBorder := colorStyle.Render("╭─ ") + title + colorStyle.Render(" "+strings.Repeat("─", dashCount)+"╮")

	// Inner content width: width minus 2 border chars and 2 padding chars
	innerWidth := width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	bodyStyle := lipgloss.NewStyle().
		Width(innerWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true).
		BorderTop(false).
		BorderForeground(borderColor).
		PaddingLeft(1).
		PaddingRight(1)

	if height > 0 {
		// height total = 1 (top border line) + inner content + 1 (bottom border)
		bodyStyle = bodyStyle.Height(height - 2)
	}

	body := bodyStyle.Render(content)
	return topBorder + "\n" + body
}

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
