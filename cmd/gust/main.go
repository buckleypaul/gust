package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/pages"
	"github.com/buckleypaul/gust/internal/west"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ws := west.DetectWorkspace(cwd)
	if ws == nil {
		fmt.Fprintln(os.Stderr, "Not in a Zephyr workspace (no west.yml found)")
		os.Exit(1)
	}

	cfg := config.Load(ws.Root)

	pageMap := map[app.PageID]app.Page{
		app.WorkspacePage: pages.NewWorkspacePage(ws),
		app.BuildPage:     pages.NewBuildPage(),
		app.FlashPage:     pages.NewFlashPage(),
		app.MonitorPage:   pages.NewMonitorPage(),
		app.TestPage:      pages.NewTestPage(),
		app.ArtifactsPage: pages.NewArtifactsPage(),
		app.WestPage:      pages.NewWestPage(),
		app.ConfigPage:    pages.NewConfigPage(),
		app.SettingsPage:  pages.NewSettingsPage(&cfg, ws.Root),
	}

	model := app.New(pageMap)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
