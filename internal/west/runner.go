package west

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandOutputMsg is sent for each line of output from a running command.
type CommandOutputMsg struct {
	Line string
}

// CommandCompletedMsg is sent when a command finishes.
type CommandCompletedMsg struct {
	ExitCode int
	Duration time.Duration
	Err      error
}

// RunStreaming executes a command and returns all output when complete.
// Note: Despite the name, this currently batches output for simplicity.
// True line-by-line streaming would require subscription-based architecture.
func RunStreaming(name string, args ...string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		cmd := exec.Command(name, args...)
		applyEnv(cmd)

		output, err := cmd.CombinedOutput()
		duration := time.Since(start)

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return CommandResultMsg{
					Output:   string(output),
					ExitCode: -1,
					Duration: duration,
				}
			}
		}

		return CommandResultMsg{
			Output:   string(output),
			ExitCode: exitCode,
			Duration: duration,
		}
	}
}

// CommandResultMsg bundles all output from a non-streaming command.
type CommandResultMsg struct {
	Output   string
	ExitCode int
	Duration time.Duration
}

// StreamCommand starts a command and sends output to a channel for real-time display.
// The caller should use tea.Program.Send() in a goroutine to forward messages.
func StreamCommand(p *tea.Program, name string, args ...string) {
	start := time.Now()
	cmd := exec.Command(name, args...)
	applyEnv(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.Send(CommandCompletedMsg{ExitCode: -1, Err: err})
		return
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		p.Send(CommandCompletedMsg{ExitCode: -1, Err: err})
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		p.Send(CommandOutputMsg{Line: scanner.Text()})
	}

	exitCode := 0
	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	p.Send(CommandCompletedMsg{
		ExitCode: exitCode,
		Duration: time.Since(start),
		Err:      err,
	})
}

// RunSimple executes a command and returns the output as a single string.
func RunSimple(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	applyEnv(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s: %w", name, err)
	}
	return string(output), nil
}
