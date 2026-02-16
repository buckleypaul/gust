package west

import tea "github.com/charmbracelet/bubbletea"

// Status runs `west status` and returns the output.
func Status() tea.Cmd {
	return RunStreaming("west", "status")
}

// List runs `west list` and returns the output.
func List() tea.Cmd {
	return RunStreaming("west", "list")
}

// Diff runs `west diff` and returns the output.
func Diff() tea.Cmd {
	return RunStreaming("west", "diff")
}

// Forall runs `west forall -c <cmd>` and returns the output.
func Forall(cmd string) tea.Cmd {
	return RunStreaming("west", "forall", "-c", cmd)
}

// Update runs `west update` and streams output.
func Update() tea.Cmd {
	return RunStreaming("west", "update")
}
