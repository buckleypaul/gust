package west

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWithRequestIDTagsCommandResult(t *testing.T) {
	cmd := WithRequestID("req-123", func() tea.Msg {
		return CommandResultMsg{Output: "ok", ExitCode: 0}
	})
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()
	result, ok := msg.(CommandResultMsg)
	if !ok {
		t.Fatalf("expected CommandResultMsg, got %T", msg)
	}
	if result.RequestID != "req-123" {
		t.Fatalf("expected request ID req-123, got %q", result.RequestID)
	}
}

func TestWithRequestIDPassesThroughNonResult(t *testing.T) {
	type customMsg struct {
		Name string
	}

	cmd := WithRequestID("req-123", func() tea.Msg {
		return customMsg{Name: "unchanged"}
	})
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()
	got, ok := msg.(customMsg)
	if !ok {
		t.Fatalf("expected customMsg, got %T", msg)
	}
	if got.Name != "unchanged" {
		t.Fatalf("expected passthrough payload, got %q", got.Name)
	}
}
