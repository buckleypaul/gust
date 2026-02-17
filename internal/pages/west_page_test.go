package pages

import (
	"testing"

	"github.com/buckleypaul/gust/internal/west"
	tea "github.com/charmbracelet/bubbletea"
)

func TestWestPageEnterUsesInjectedRunner(t *testing.T) {
	fake := &fakeRunner{
		nextMsg: west.CommandResultMsg{Output: "status"},
	}
	p := NewWestPage(fake)

	page, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := page.(*WestPage)
	if cmd == nil {
		t.Fatal("expected command")
	}
	if !updated.running {
		t.Fatal("expected running state")
	}
	if fake.statusCalls != 1 {
		t.Fatalf("expected status command to be invoked once, got %d", fake.statusCalls)
	}

	msg := cmd()
	result, ok := msg.(west.CommandResultMsg)
	if !ok {
		t.Fatalf("expected CommandResultMsg, got %T", msg)
	}
	if result.RequestID == "" {
		t.Fatal("expected request ID")
	}
}
