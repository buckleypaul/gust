package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

// flashSection holds per-flash state for the combined Project page.
// It is not a Page; ProjectPage orchestrates it.
type flashSection struct {
	flashing   bool
	flashStart time.Time
	lastBuild  *store.BuildRecord
	message    string
	seq        int
}

func (f *flashSection) nextRequestID() string {
	f.seq++
	return fmt.Sprintf("flash-%d", f.seq)
}

func (f *flashSection) refreshLastBuild(s *store.Store) {
	if s == nil {
		f.lastBuild = nil
		return
	}
	builds, err := s.Builds()
	if err != nil || len(builds) == 0 {
		f.lastBuild = nil
		return
	}
	last := builds[len(builds)-1]
	f.lastBuild = &last
}

// viewSection renders the Flash section header and status.
func (f *flashSection) viewSection(width int) string {
	var sb strings.Builder
	sectionLabel := lipgloss.NewStyle().Foreground(ui.Subtle).Bold(true)
	separator := strings.Repeat("─", max(width-9, 10))
	sb.WriteString("  " + sectionLabel.Render("── Flash "+separator) + "\n")

	if f.lastBuild != nil {
		ts := f.lastBuild.Timestamp.Format("Jan 02 15:04")
		if f.lastBuild.Success {
			sb.WriteString("  Last build: " + ui.SuccessBadge("OK") + fmt.Sprintf("  (%s)\n", ts))
		} else {
			sb.WriteString("  Last build: " + ui.ErrorBadge("FAILED") + fmt.Sprintf("  (%s)\n", ts))
		}
	} else {
		sb.WriteString("  " + ui.DimStyle.Render("No recent builds. Run a build first.") + "\n")
	}
	if f.message != "" {
		sb.WriteString("  " + f.message + "\n")
	}
	if f.flashing {
		sb.WriteString("  " + ui.DimStyle.Render("Flashing...") + "\n")
	}
	return sb.String()
}

// start launches west flash, writes the command header to out.
func (f *flashSection) start(buildDir, flashRunner string, runner west.Runner, out *strings.Builder) (requestID string, cmd tea.Cmd) {
	f.flashing = true
	f.flashStart = time.Now()
	f.message = ""
	requestID = f.nextRequestID()

	args := []string{"flash"}
	if buildDir != "" {
		args = append(args, "-d", buildDir)
	}
	if flashRunner != "" {
		args = append(args, "--runner", flashRunner)
	}
	out.WriteString("$ west " + strings.Join(args, " ") + "\n\n")
	return requestID, west.WithRequestID(requestID, runner.Run("west", args...))
}

// complete finalises flash state and records to store.
func (f *flashSection) complete(result west.CommandResultMsg, board string, s *store.Store, out *strings.Builder) {
	f.flashing = false
	success := result.ExitCode == 0
	out.WriteString(result.Output)
	status := "success"
	if !success {
		status = fmt.Sprintf("failed (exit code: %d)", result.ExitCode)
	}
	out.WriteString(fmt.Sprintf("\nFlash %s in %s\n", status, result.Duration))
	if s != nil {
		_ = s.AddFlash(store.FlashRecord{
			Board:     board,
			Timestamp: f.flashStart,
			Success:   success,
			Duration:  result.Duration.String(),
		})
	}
}
