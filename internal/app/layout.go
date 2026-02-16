package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/ui"
)

const sidebarWidth = 22 // 20 content + 2 border/padding

func renderSidebar(pages []PageID, active PageID, pageMap map[PageID]Page, height int, focused bool) string {
	var b strings.Builder
	title := "gust"
	if focused {
		title = ui.BoldStyle.Render("gust [FOCUSED]")
	} else {
		title = ui.TitleStyle.Render("gust")
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	for i, id := range pages {
		p := pageMap[id]
		num := fmt.Sprintf("%d ", i+1)
		if id == active {
			b.WriteString(ui.SidebarActiveStyle.Render("▸ " + num + p.Name()))
		} else {
			b.WriteString(ui.SidebarItemStyle.Render("  " + num + p.Name()))
		}
		b.WriteString("\n")
	}

	style := ui.SidebarStyle.Height(height)
	if focused {
		style = style.BorderForeground(ui.Primary)
	}
	return style.Render(b.String())
}

func renderStatusBar(pageHelp []key.Binding, width int, focus FocusArea) string {
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
		ui.StatusKey("1-9", "jump"),
		ui.StatusKey("?", "help"),
		ui.StatusKey("q", "quit"),
	)

	line := strings.Join(parts, "  ")
	return ui.StatusBarStyle.Width(width).Render(line)
}

func renderLayout(sidebar, content, statusBar string) string {
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, main, statusBar)
}
