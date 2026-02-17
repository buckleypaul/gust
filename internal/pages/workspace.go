package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type setupStep int

const (
	stepBrewDeps setupStep = iota
	stepInit
	stepUpdate
	stepExport
	stepPipInstall
	stepSdkInstall
	stepCount
)

var stepLabels = [stepCount]string{
	"Install system dependencies (Homebrew)",
	"Initialize workspace (west init)",
	"Update modules (west update)",
	"Export CMake packages (west zephyr-export)",
	"Install Python dependencies",
	"Install Zephyr SDK",
}

type WorkspacePage struct {
	workspace     *west.Workspace
	health        west.WorkspaceHealth
	updating      bool
	output        strings.Builder
	viewport      viewport.Model
	spinner       spinner.Model
	width, height int
	message       string

	settingUp   bool
	currentStep setupStep
	stepsDone   [stepCount]bool
	setupFailed bool
}

func NewWorkspacePage(ws *west.Workspace) *WorkspacePage {
	vp := viewport.New(0, 0)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = ui.AccentStyle

	page := &WorkspacePage{
		workspace: ws,
		viewport:  vp,
		spinner:   s,
	}

	// Initial health check
	if ws != nil {
		page.health = ws.CheckHealth()
	}

	return page
}

func (p *WorkspacePage) Init() tea.Cmd { return p.spinner.Tick }

func (p *WorkspacePage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case spinner.TickMsg:
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd

	case tea.KeyMsg:
		if p.updating || p.settingUp {
			p.viewport, cmd = p.viewport.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "s":
			return p, p.startSetup()
		case "u":
			if p.workspace != nil && p.workspace.Initialized {
				p.updating = true
				p.output.Reset()
				p.output.WriteString("Running west update...\n\n")
				p.viewport.SetContent(p.output.String())
				return p, west.Update()
			}
		case "c":
			p.output.Reset()
			p.viewport.SetContent("")
			p.message = ""
			p.settingUp = false
			p.setupFailed = false
			p.stepsDone = [stepCount]bool{}
		case "r":
			// Refresh health status
			if p.workspace != nil {
				p.health = p.workspace.CheckHealth()
				p.message = "Status refreshed"
			}
		}

	case west.CommandResultMsg:
		if p.settingUp {
			return p, p.handleSetupResult(msg)
		}

		// Only handle command results if we're actually running an update
		if !p.updating {
			return p, nil
		}

		p.updating = false
		p.output.WriteString(msg.Output)
		if msg.ExitCode == 0 {
			p.message = "Update completed successfully"
		} else {
			p.message = fmt.Sprintf("Update failed (exit code: %d)", msg.ExitCode)
		}
		p.output.WriteString(fmt.Sprintf("\n%s in %s\n", p.message, msg.Duration))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()
		return p, nil
	}

	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *WorkspacePage) startSetup() tea.Cmd {
	p.settingUp = true
	p.setupFailed = false
	p.stepsDone = [stepCount]bool{}
	p.output.Reset()
	p.message = ""

	// Always start with brew deps check (step 0)
	p.currentStep = stepBrewDeps

	// Mark west init as done if workspace already initialized
	if p.workspace != nil && p.workspace.Initialized {
		p.stepsDone[stepInit] = true
	}

	return p.startStep()
}

func (p *WorkspacePage) startStep() tea.Cmd {
	label := stepLabels[p.currentStep]
	p.output.WriteString(fmt.Sprintf("=== %s ===\n", label))
	p.output.WriteString("Running...\n\n")
	p.viewport.SetContent(p.output.String())
	p.viewport.GotoBottom()

	switch p.currentStep {
	case stepBrewDeps:
		return west.InstallBrewDeps()
	case stepInit:
		return west.Init()
	case stepUpdate:
		return west.Update()
	case stepExport:
		return west.ZephyrExport()
	case stepPipInstall:
		return west.PackagesPipInstall()
	case stepSdkInstall:
		return west.SdkInstall()
	}
	return nil
}

func (p *WorkspacePage) handleSetupResult(msg west.CommandResultMsg) tea.Cmd {
	p.output.WriteString(msg.Output)

	if msg.ExitCode != 0 {
		p.setupFailed = true
		p.settingUp = false
		p.message = fmt.Sprintf("Setup failed at: %s (exit code: %d, %s)",
			stepLabels[p.currentStep], msg.ExitCode, msg.Duration)
		p.output.WriteString(fmt.Sprintf("\n%s\n", p.message))
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		// Refresh health after failure
		if p.workspace != nil {
			p.health = p.workspace.CheckHealth()
		}
		return nil
	}

	p.output.WriteString(fmt.Sprintf("  Completed in %s\n\n", msg.Duration))
	p.stepsDone[p.currentStep] = true

	// Find next step that isn't already done
	next := p.currentStep + 1
	for next < stepCount && p.stepsDone[next] {
		// Skip steps already marked as done
		p.output.WriteString(fmt.Sprintf("=== %s ===\n", stepLabels[next]))
		p.output.WriteString("  Skipped (already done)\n\n")
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()
		next++
	}

	if next >= stepCount {
		p.settingUp = false
		p.message = "Setup completed successfully!"
		p.output.WriteString(p.message + "\n")
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		if p.workspace != nil {
			p.workspace.Initialized = true
			p.workspace.ManifestPath = west.ResolveManifest(p.workspace.Root)
			// Refresh health after completion
			p.health = p.workspace.CheckHealth()
		}
		return nil
	}

	p.currentStep = next
	return p.startStep()
}

func (p *WorkspacePage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Workspace"))
	b.WriteString("\n")

	ws := p.workspace
	if ws == nil {
		b.WriteString("  No workspace detected.\n")
		return b.String()
	}

	// Show setup progress during setup
	if p.settingUp {
		b.WriteString("  Setting up...\n\n")
		b.WriteString(p.renderSetupChecklist())
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
		return b.String()
	}

	if p.setupFailed {
		b.WriteString("  " + ui.ErrorBadge("Setup Failed") + "\n\n")
		b.WriteString(p.renderSetupChecklist())
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
		return b.String()
	}

	// Determine overall status badge
	statusBadge := p.getStatusBadge()
	b.WriteString("  " + statusBadge + "\n\n")
	b.WriteString(fmt.Sprintf("  Root:     %s\n", ws.Root))
	b.WriteString(fmt.Sprintf("  Manifest: %s\n", ws.ManifestPath))

	// Always show health checklist
	b.WriteString("\n  Setup Status:\n")
	b.WriteString(p.renderHealthChecklist())

	// Show actionable guidance if setup is incomplete
	if !p.isFullySetup() {
		b.WriteString("\n  " + ui.ErrorBadge("Action Required") + "\n")
		if !ws.Initialized {
			b.WriteString("  Press 's' to run full setup wizard\n")
		} else {
			b.WriteString("  Press 's' to complete missing setup steps\n")
		}
	}

	if p.message != "" {
		b.WriteString("\n  " + p.message + "\n")
	}

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
	}

	return b.String()
}

// getStatusBadge returns the appropriate status badge based on workspace health
func (p *WorkspacePage) getStatusBadge() string {
	if !p.workspace.Initialized {
		return ui.ErrorBadge("Not Initialized")
	}

	if p.isFullySetup() {
		return ui.SuccessBadge("Ready")
	}

	return ui.ErrorBadge("Incomplete Setup")
}

// isFullySetup checks if all required components are ready
func (p *WorkspacePage) isFullySetup() bool {
	return p.health.BrewDepsOK &&
		p.health.WestInitialized &&
		p.health.ModulesUpdated &&
		p.health.ZephyrExported &&
		p.health.PythonDepsOK &&
		p.health.SdkInstalled
}

// renderSetupChecklist shows setup progress during the setup wizard
func (p *WorkspacePage) renderSetupChecklist() string {
	var b strings.Builder
	for i := 0; i < int(stepCount); i++ {
		step := setupStep(i)
		var marker string
		var label string
		switch {
		case p.stepsDone[step]:
			marker = "[✓]"
			label = stepLabels[step]
		case p.settingUp && step == p.currentStep:
			// Show spinner for currently running step
			marker = p.spinner.View()
			label = stepLabels[step] + " (running...)"
		default:
			marker = "[ ]"
			label = stepLabels[step]
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", marker, label))
	}
	return b.String()
}

// renderHealthChecklist shows the actual health status of workspace components
func (p *WorkspacePage) renderHealthChecklist() string {
	var b strings.Builder

	checks := []struct {
		label  string
		status bool
	}{
		{"System dependencies (Homebrew)", p.health.BrewDepsOK},
		{"West workspace initialized", p.health.WestInitialized},
		{"Zephyr modules updated", p.health.ModulesUpdated},
		{"CMake packages exported", p.health.ZephyrExported},
		{"Python dependencies installed", p.health.PythonDepsOK},
		{"Zephyr SDK installed", p.health.SdkInstalled},
	}

	for _, check := range checks {
		marker := "[ ]"
		if check.status {
			marker = "[✓]"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", marker, check.label))
	}

	return b.String()
}

func (p *WorkspacePage) Name() string { return "Workspace" }

func (p *WorkspacePage) ShortHelp() []key.Binding {
	if p.settingUp {
		return []key.Binding{
			key.NewBinding(key.WithKeys(""), key.WithHelp("", "setup in progress...")),
		}
	}

	if p.workspace != nil && p.workspace.Initialized {
		return []key.Binding{
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "run setup")),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "west update")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh status")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		}
	}

	return []key.Binding{
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "setup")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh status")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *WorkspacePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	// Reserve space for header content above viewport:
	// - Title, status badge, root/manifest info
	// - Health checklist (6 items + label)
	// - Action required message
	// - Completion message
	// Total: ~25 lines
	vpHeight := h - 25
	if vpHeight < 10 {
		vpHeight = 10
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}
