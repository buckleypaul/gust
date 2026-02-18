package pages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type formField int

const (
	fieldPristine formField = iota
	fieldCMakeArgs
	fieldCount
)

type buildState int

const (
	buildStateIdle buildState = iota
	buildStateRunning
	buildStateDone
)

const (
	labelWidth       = 11 // "Project  " padded
	minLeftWidth     = 30
	maxLeftWidth     = 50
	maxDropdownItems = 10
)

// buildSection holds per-build state for the combined Project page.
// It is not a Page; ProjectPage orchestrates it.
type buildSection struct {
	cmakeInput textinput.Model
	pristine   bool
	state      buildState
	buildStart time.Time
	gitBranch  string
	gitCommit  string
	gitDirty   bool
	message    string
	seq        int
}

func newBuildSection() buildSection {
	cmake := textinput.New()
	cmake.Placeholder = "e.g. -DOVERLAY_CONFIG=overlay.conf"
	cmake.CharLimit = 512
	cmake.Prompt = ""
	return buildSection{cmakeInput: cmake}
}

func (b *buildSection) nextRequestID() string {
	b.seq++
	return fmt.Sprintf("build-%d", b.seq)
}

// viewSection renders the Build section header and controls.
func (b *buildSection) viewSection(width int, focusedCMake bool) string {
	var sb strings.Builder
	sectionLabel := lipgloss.NewStyle().Foreground(ui.Subtle).Bold(true)
	separator := strings.Repeat("─", max(width-9, 10))
	sb.WriteString("  " + sectionLabel.Render("── Build "+separator) + "\n")

	focusedLabel := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)
	normalLabel := lipgloss.NewStyle().Foreground(ui.Text)

	check := "[ ]"
	if b.pristine {
		check = "[x]"
	}
	sb.WriteString("  " + normalLabel.Render(fmt.Sprintf("%-9s", "Pristine")) + " " + check + "\n")

	inputWidth := width - labelWidth - 4
	if inputWidth < 10 {
		inputWidth = 10
	}
	b.cmakeInput.Width = inputWidth
	lbl := normalLabel.Render(fmt.Sprintf("%-9s", "CMake"))
	if focusedCMake {
		lbl = focusedLabel.Render(fmt.Sprintf("%-9s", "CMake"))
	}
	sb.WriteString("  " + lbl + " " + b.cmakeInput.View() + "\n")

	if b.message != "" {
		sb.WriteString("  " + b.message + "\n")
	}
	if b.state == buildStateRunning {
		sb.WriteString("  " + ui.DimStyle.Render("Building...") + "\n")
	}
	return sb.String()
}

// start launches west build, writes the command header to out, and returns
// the request ID and the tea.Cmd to execute.
func (b *buildSection) start(wsRoot, project, board, shield, buildDir string, runner west.Runner, out *strings.Builder) (requestID string, cmd tea.Cmd) {
	b.state = buildStateRunning
	b.buildStart = time.Now()
	b.message = ""
	requestID = b.nextRequestID()

	if !filepath.IsAbs(project) {
		project = filepath.Join(wsRoot, project)
	}

	b.gitBranch, b.gitCommit, b.gitDirty = "", "", false
	gitDir := project
	if gitDir == "" {
		gitDir = wsRoot
	}
	if o, err := gitCmd(gitDir, "branch", "--show-current"); err == nil {
		b.gitBranch = strings.TrimSpace(o)
	}
	if o, err := gitCmd(gitDir, "rev-parse", "--short=8", "HEAD"); err == nil {
		b.gitCommit = strings.TrimSpace(o)
	}
	if o, err := gitCmd(gitDir, "status", "--porcelain"); err == nil {
		b.gitDirty = strings.TrimSpace(o) != ""
	}

	args := []string{"build", "-b", board}
	if buildDir != "" {
		args = append(args, "-d", buildDir)
	}
	if b.pristine {
		args = append(args, "-p", "always")
	}
	if shield != "" {
		args = append(args, "--shield", shield)
	}
	if cmake := b.cmakeInput.Value(); cmake != "" {
		args = append(args, "--")
		args = append(args, strings.Fields(cmake)...)
	}
	args = append(args, project)

	out.WriteString("$ west " + strings.Join(args, " ") + "\n\n")
	return requestID, west.WithRequestID(requestID, runner.Run("west", args...))
}

// complete finalises build state and records to store.
func (b *buildSection) complete(result west.CommandResultMsg, board, app, shield, buildDir string, s *store.Store, wsRoot string, out *strings.Builder) {
	b.state = buildStateDone
	success := result.ExitCode == 0
	out.WriteString(result.Output)
	status := "success"
	if !success {
		status = fmt.Sprintf("failed (exit code: %d)", result.ExitCode)
	}
	out.WriteString(fmt.Sprintf("\nBuild %s in %s\n", status, result.Duration))

	var binarySize int64
	if success {
		dir := buildDir
		if dir == "" {
			dir = "build"
		}
		if fi, err := os.Stat(filepath.Join(wsRoot, dir, "zephyr", "zephyr.bin")); err == nil {
			binarySize = fi.Size()
		}
	}
	if s != nil {
		_ = s.AddBuild(store.BuildRecord{
			Board:      board,
			App:        app,
			Timestamp:  b.buildStart,
			Success:    success,
			Duration:   result.Duration.String(),
			Shield:     shield,
			Pristine:   b.pristine,
			CMakeArgs:  b.cmakeInput.Value(),
			GitBranch:  b.gitBranch,
			GitCommit:  b.gitCommit,
			GitDirty:   b.gitDirty,
			BuildDir:   buildDir,
			BinarySize: binarySize,
		})
	}
}

// gitCmd runs a git subcommand in dir and returns stdout.
func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
