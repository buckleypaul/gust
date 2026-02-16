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

// RunStreaming executes a command and streams output line-by-line as tea.Msg.
// It returns a tea.Cmd that, when run by bubbletea, sends CommandOutputMsg
// for each line and a final CommandCompletedMsg.
func RunStreaming(name string, args ...string) tea.Cmd {
	return func() tea.Msg {
		// We can't stream from inside a single tea.Cmd return,
		// so we'll collect all output and return a batch result.
		// For true streaming, we'll use a channel-based approach
		// via the program's Send method (done in the page layer).
		// For now, return a collected result.
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
				return CommandCompletedMsg{ExitCode: -1, Duration: duration, Err: err}
			}
		}

		// Return a combined message with all output
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
