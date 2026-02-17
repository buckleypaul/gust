package pages

import (
	"errors"
	"strings"
	"testing"
)

func TestMonitorPageAppliesConnectedStateFromMessage(t *testing.T) {
	p := NewMonitorPage(nil, 115200)

	page, cmd := p.Update(monitorConnectedMsg{
		portName: "tty.usbmodem123",
		baudRate: 115200,
	})
	updated := page.(*MonitorPage)

	if updated.state != monitorStateConnected {
		t.Fatalf("expected connected state, got %v", updated.state)
	}
	if !updated.input.Focused() {
		t.Fatal("expected input to be focused")
	}
	if !strings.Contains(updated.message, "Connected to tty.usbmodem123 @ 115200") {
		t.Fatalf("unexpected status message: %q", updated.message)
	}
	if cmd == nil {
		t.Fatal("expected follow-up command to be scheduled")
	}
}

func TestMonitorPageConnectErrorUpdatesMessage(t *testing.T) {
	p := NewMonitorPage(nil, 115200)

	page, _ := p.Update(monitorConnectedMsg{err: errors.New("permission denied")})
	updated := page.(*MonitorPage)

	if updated.state != monitorStatePortSelect {
		t.Fatalf("expected to remain in port select state, got %v", updated.state)
	}
	if !strings.Contains(updated.message, "Failed to connect: permission denied") {
		t.Fatalf("unexpected status message: %q", updated.message)
	}
}
