package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
	"github.com/buckleypaul/gust/internal/west"
)

type setupStep int

const (
	stepInit setupStep = iota
	stepUpdate
	stepExport
	stepPipInstall
	stepSdkInstall
	stepCount
)

var stepLabels = [stepCount]string{
	"Initialize workspace (west init)",
	"Update modules (west update)",
	"Export CMake packages (west zephyr-export)",
	"Install Python dependencies",
	"Install Zephyr SDK",
}

type WorkspacePage struct {
	workspace     *west.Workspace
	updating      bool
	output        strings.Builder
	viewport      viewport.Model
	width, height int
	message       string

	settingUp   bool
	currentStep setupStep
	stepsDone   [stepCount]bool
	setupFailed bool
}

func NewWorkspacePage(ws *west.Workspace) *WorkspacePage {
	vp := viewport.New(0, 0)
	return &WorkspacePage{
		workspace: ws,
		viewport:  vp,
	}
}

func (p *WorkspacePage) Init() tea.Cmd { return nil }

func (p *WorkspacePage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.updating || p.settingUp {
			var cmd tea.Cmd
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
		}

	case west.CommandResultMsg:
		if p.settingUp {
			return p, p.handleSetupResult(msg)
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

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *WorkspacePage) startSetup() tea.Cmd {
	p.settingUp = true
	p.setupFailed = false
	p.stepsDone = [stepCount]bool{}
	p.output.Reset()
	p.message = ""

	if p.workspace != nil && p.workspace.Initialized {
		p.currentStep = stepUpdate
		p.stepsDone[stepInit] = true
	} else {
		p.currentStep = stepInit
	}

	return p.startStep()
}

func (p *WorkspacePage) startStep() tea.Cmd {
	label := stepLabels[p.currentStep]
	p.output.WriteString(fmt.Sprintf("=== %s ===\n", label))
	p.viewport.SetContent(p.output.String())
	p.viewport.GotoBottom()

	switch p.currentStep {
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
		return nil
	}

	p.output.WriteString(fmt.Sprintf("  Completed in %s\n\n", msg.Duration))
	p.stepsDone[p.currentStep] = true

	next := p.currentStep + 1
	if next >= stepCount {
		p.settingUp = false
		p.message = "Setup completed successfully!"
		p.output.WriteString(p.message + "\n")
		p.viewport.SetContent(p.output.String())
		p.viewport.GotoBottom()

		if p.workspace != nil {
			p.workspace.Initialized = true
			p.workspace.ManifestPath = west.ResolveManifest(p.workspace.Root)
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

	if p.settingUp {
		b.WriteString("  Setting up...\n\n")
		b.WriteString(p.renderChecklist())
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
		return b.String()
	}

	if p.setupFailed {
		b.WriteString("  " + ui.ErrorBadge("Setup Failed") + "\n\n")
		b.WriteString(p.renderChecklist())
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
		return b.String()
	}

	if !ws.Initialized {
		b.WriteString("  " + ui.ErrorBadge("Not Initialized") + "\n\n")
		b.WriteString("  Workspace found but not initialized.\n\n")
		b.WriteString("  Setup steps:\n")
		b.WriteString(p.renderChecklist())
		b.WriteString("\n  Press 's' to start setup\n")
		return b.String()
	}

	b.WriteString("  " + ui.SuccessBadge("Initialized") + "\n\n")
	b.WriteString(fmt.Sprintf("  Root:     %s\n", ws.Root))
	b.WriteString(fmt.Sprintf("  Manifest: %s\n", ws.ManifestPath))

	if p.message != "" {
		b.WriteString("\n  " + p.message + "\n")
	}

	if p.output.Len() > 0 {
		b.WriteString("\n")
		b.WriteString(p.viewport.View())
	}

	return b.String()
}

func (p *WorkspacePage) renderChecklist() string {
	var b strings.Builder
	for i := 0; i < int(stepCount); i++ {
		step := setupStep(i)
		var marker string
		switch {
		case p.stepsDone[step]:
			marker = "[✓]"
		case p.settingUp && step == p.currentStep:
			marker = "[►]"
		default:
			marker = "[ ]"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", marker, stepLabels[step]))
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
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "west update")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "run setup steps")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		}
	}

	return []key.Binding{
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "setup")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
	}
}

func (p *WorkspacePage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h - 10
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}
