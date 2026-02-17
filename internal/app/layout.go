package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/ui"
)

const sidebarWidth = 26 // 20 content + 2 padding + 2 border + 2 extra

func renderProjectBar(selectedProject, selectedBoard string, width int) string {
	projectDisplay := selectedProject
	if projectDisplay == "" {
		projectDisplay = "(none)"
	}
	boardDisplay := selectedBoard
	if boardDisplay == "" {
		boardDisplay = "(none)"
	}

	left := "  Project: " + projectDisplay
	right := "Board: " + boardDisplay + "  "

	// Pad between left and right zones
	gap := width - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	content := left + strings.Repeat(" ", gap) + right
	return ui.StatusBarStyle.Width(width).Render(content)
}

func renderSidebar(pages []PageID, active PageID, pageMap map[PageID]Page, height int, focused bool) string {
	var b strings.Builder

	for _, id := range pages {
		p := pageMap[id]
		if id == active {
			b.WriteString(ui.SidebarActiveStyle.Render("▸ " + p.Name()))
		} else {
			b.WriteString(ui.SidebarItemStyle.Render("  " + p.Name()))
		}
		b.WriteString("\n")
	}

	return ui.Panel("gust", b.String(), sidebarWidth, height, focused)
}

func renderStatusBar(pageHelp []key.Binding, width int, focus FocusArea, wsRoot string) string {
	var parts []string

	// Focus-specific instructions
	if focus == FocusSidebar {
		parts = append(parts,
			ui.StatusKey("↑/↓", "navigate"),
			ui.StatusKey("enter", "select"),
		)
	} else {
		// Page-specific keys when content is focused
		for _, kb := range pageHelp {
			if kb.Enabled() {
				parts = append(parts, ui.StatusKey(kb.Help().Key, kb.Help().Desc))
			}
		}
	}

	// Always add global keys
	parts = append(parts,
		ui.StatusKey("tab", "focus"),
		ui.StatusKey("?", "help"),
		ui.StatusKey("q", "quit"),
	)

	left := strings.Join(parts, "  ")

	// Right zone: truncated workspace path
	wsDisplay := wsRoot
	if len(wsDisplay) > 22 {
		wsDisplay = "…" + wsDisplay[len(wsDisplay)-21:]
	}
	right := wsDisplay

	// Build left+right layout
	leftRendered := ui.StatusBarStyle.Render(left)
	rightRendered := ui.StatusBarStyle.Render(right)

	leftWidth := lipgloss.Width(leftRendered)
	rightWidth := lipgloss.Width(rightRendered)
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	filler := ui.StatusBarStyle.Width(gap).Render("")
	return leftRendered + filler + rightRendered
}

func renderLayout(projectBar, sidebar, content, statusBar string) string {
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, projectBar, main, statusBar)
}
