package pages

import (
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/west"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkspacePageIgnoresForeignUpdateResult(t *testing.T) {
	p := NewWorkspacePage(nil)
	p.updating = true
	p.activeRequestID = "workspace-1"

	page, _ := p.Update(west.CommandResultMsg{
		RequestID: "workspace-2",
		Output:    "foreign",
		ExitCode:  0,
	})
	updated := page.(*WorkspacePage)

	if !updated.updating {
		t.Fatal("expected update to stay running for foreign result")
	}
}

func TestWorkspacePageHandlesMatchingUpdateResult(t *testing.T) {
	p := NewWorkspacePage(nil)
	p.updating = true
	p.activeRequestID = "workspace-1"

	page, _ := p.Update(west.CommandResultMsg{
		RequestID: "workspace-1",
		Output:    "done",
		ExitCode:  0,
	})
	updated := page.(*WorkspacePage)

	if updated.updating {
		t.Fatal("expected update to stop after matching result")
	}
	if updated.activeRequestID != "" {
		t.Fatalf("expected active request to be cleared, got %q", updated.activeRequestID)
	}
}

func TestWorkspacePageStartSetupUsesInjectedRunner(t *testing.T) {
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{
			Output:   "brew deps done",
			ExitCode: 0,
			Duration: time.Second,
		},
	}
	p := NewWorkspacePage(nil, fake)

	cmd := p.startSetup()
	if cmd == nil {
		t.Fatal("expected setup command")
	}

	if fake.installBrewDepsCalls != 1 {
		t.Fatalf("expected brew deps step to run first, got %d calls", fake.installBrewDepsCalls)
	}

	msg := cmd()
	result, ok := msg.(west.CommandResultMsg)
	if !ok {
		t.Fatalf("expected CommandResultMsg, got %T", msg)
	}
	if result.RequestID == "" {
		t.Fatal("expected request ID to be attached")
	}
}

func TestWorkspacePageUpdateKeyUsesInjectedRunner(t *testing.T) {
	ws := &west.Workspace{Initialized: true}
	fake := &fakeRunner{nextMsg: west.CommandResultMsg{Output: "update", ExitCode: 0}}
	p := NewWorkspacePage(ws, fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	updated := page.(*WorkspacePage)
	if cmd == nil {
		t.Fatal("expected command")
	}
	if !updated.updating {
		t.Fatal("expected updating=true")
	}
	if fake.updateCalls != 1 {
		t.Fatalf("expected update command to use runner once, got %d", fake.updateCalls)
	}
}
