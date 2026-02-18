package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/west"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFlashPagePassesBuildDirAndRunner(t *testing.T) {
	cfg := config.Defaults()
	cfg.BuildDir = "build-custom"
	cfg.FlashRunner = "jlink"
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewFlashPage(nil, &cfg, t.TempDir(), fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
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
	if !strings.Contains(argStr, "-d build-custom") {
		t.Fatalf("expected -d build-custom in args, got %v", args)
	}
	if !strings.Contains(argStr, "--runner jlink") {
		t.Fatalf("expected --runner jlink in args, got %v", args)
	}
}

func TestFlashPageOmitsEmptyFlags(t *testing.T) {
	cfg := config.Defaults()
	cfg.BuildDir = ""
	cfg.FlashRunner = ""
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "ok",
			ExitCode: 0,
			Duration: time.Second,
		},
	}

	p := NewFlashPage(nil, &cfg, t.TempDir(), fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	_ = page
	if cmd == nil {
		t.Fatal("expected command")
	}
	_ = cmd()

	args := fake.runCalls[0].args
	if len(args) != 1 || args[0] != "flash" {
		t.Fatalf("expected bare [flash] args, got %v", args)
	}
}

func TestFlashPageHandlesBroadcastMessages(t *testing.T) {
	cfg := config.Defaults()
	p := NewFlashPage(nil, &cfg, t.TempDir())

	page, _ := p.Update(app.BoardSelectedMsg{Board: "my_board"})
	p = page.(*FlashPage)
	if p.selectedBoard != "my_board" {
		t.Fatalf("expected board my_board, got %s", p.selectedBoard)
	}

	page, _ = p.Update(app.BuildDirChangedMsg{Dir: "build-x"})
	p = page.(*FlashPage)
	if p.buildDir != "build-x" {
		t.Fatalf("expected buildDir build-x, got %s", p.buildDir)
	}

	page, _ = p.Update(app.FlashRunnerChangedMsg{Runner: "openocd"})
	p = page.(*FlashPage)
	if p.flashRunner != "openocd" {
		t.Fatalf("expected runner openocd, got %s", p.flashRunner)
	}
}
