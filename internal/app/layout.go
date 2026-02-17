package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/ui"
)

const sidebarWidth = 22 // 20 content + 2 border/padding

func renderProjectBar(selectedProject, selectedBoard string, width int, sidebarFocused bool) string {
	projectDisplay := selectedProject
	if projectDisplay == "" {
		projectDisplay = "(none)"
	}
	boardDisplay := selectedBoard
	if boardDisplay == "" {
		boardDisplay = "(none)"
	}
	content := fmt.Sprintf("Project: %s  Board: %s", projectDisplay, boardDisplay)
	hint := ""
	if sidebarFocused {
		hint = ui.DimStyle.Render("  [p] change")
	}
	return ui.StatusBarStyle.Width(width).Render(content + hint)
}

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

	for _, id := range pages {
		p := pageMap[id]
		if id == active {
			b.WriteString(ui.SidebarActiveStyle.Render("▸ " + p.Name()))
		} else {
			b.WriteString(ui.SidebarItemStyle.Render("  " + p.Name()))
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
			ui.StatusKey("p", "project"),
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

	line := strings.Join(parts, "  ")
	return ui.StatusBarStyle.Width(width).Render(line)
}

func renderLayout(projectBar, sidebar, content, statusBar string) string {
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, projectBar, main, statusBar)
}
