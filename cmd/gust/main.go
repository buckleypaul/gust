package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/pages"
	"github.com/buckleypaul/gust/internal/store"
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
	st := store.New(filepath.Join(ws.Root, ".gust"))

	pageMap := map[app.PageID]app.Page{
		app.WorkspacePage: pages.NewWorkspacePage(ws),
		app.BuildPage:     pages.NewBuildPage(st),
		app.FlashPage:     pages.NewFlashPage(st),
		app.MonitorPage:   pages.NewMonitorPage(st, cfg.SerialBaudRate),
		app.TestPage:      pages.NewTestPage(st),
		app.ArtifactsPage: pages.NewArtifactsPage(st),
		app.WestPage:      pages.NewWestPage(),
		app.ConfigPage:    pages.NewConfigPage(ws.Root),
		app.SettingsPage:  pages.NewSettingsPage(&cfg, ws.Root),
	}

	model := app.New(pageMap)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
