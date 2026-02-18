package pages

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/west"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTestPagePassesBoardAndBuildDir(t *testing.T) {
	wsRoot := t.TempDir()
	cfg := config.Defaults()
	cfg.DefaultBoard = "nrf52840dk_nrf52840"
	cfg.BuildDir = "build-test"
	cfg.LastProject = filepath.Join("apps", "demo")
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewTestPage(nil, &cfg, wsRoot, fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	_ = page
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	if len(fake.runCalls) != 1 {
		t.Fatalf("expected 1 run call, got %d", len(fake.runCalls))
	}
	args := fake.runCalls[0].args
	argStr := strings.Join(args, " ")
	if !strings.Contains(argStr, "-b nrf52840dk_nrf52840") {
		t.Fatalf("expected -b flag in args, got %v", args)
	}
	if !strings.Contains(argStr, "-d build-test") {
		t.Fatalf("expected -d build-test in args, got %v", args)
	}
	expectedProject := filepath.Join(wsRoot, "apps", "demo")
	if !strings.Contains(argStr, expectedProject) {
		t.Fatalf("expected project path %s in args, got %v", expectedProject, args)
	}
}

func TestTestPageOmitsEmptyTargeting(t *testing.T) {
	cfg := config.Defaults()
	cfg.DefaultBoard = ""
	cfg.BuildDir = ""
	cfg.LastProject = ""
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewTestPage(nil, &cfg, t.TempDir(), fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	_ = page
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	args := fake.runCalls[0].args
	if len(args) != 3 || args[0] != "build" || args[1] != "-t" || args[2] != "run" {
		t.Fatalf("expected bare [build -t run] args, got %v", args)
	}
}

func TestTestPageHandlesBroadcastMessages(t *testing.T) {
	cfg := config.Defaults()
	p := NewTestPage(nil, &cfg, t.TempDir())

	page, _ := p.Update(app.ProjectSelectedMsg{Path: "apps/test"})
	p = page.(*TestPage)
	if p.selectedProject != "apps/test" {
		t.Fatalf("expected project apps/test, got %s", p.selectedProject)
	}

	page, _ = p.Update(app.BoardSelectedMsg{Board: "my_board"})
	p = page.(*TestPage)
	if p.selectedBoard != "my_board" {
		t.Fatalf("expected board my_board, got %s", p.selectedBoard)
	}

	page, _ = p.Update(app.BuildDirChangedMsg{Dir: "build-x"})
	p = page.(*TestPage)
	if p.buildDir != "build-x" {
		t.Fatalf("expected buildDir build-x, got %s", p.buildDir)
	}
}
