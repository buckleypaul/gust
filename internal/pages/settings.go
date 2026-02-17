package pages

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/config"
	"github.com/buckleypaul/gust/internal/ui"
)

type settingField struct {
	label string
	key   string
}

var settingFields = []settingField{
	{"Default Board", "default_board"},
	{"Serial Port", "serial_port"},
	{"Serial Baud Rate", "serial_baud_rate"},
	{"Build Directory", "build_dir"},
	{"Flash Runner", "flash_runner"},
}

type SettingsPage struct {
	cfg           *config.Config
	workspaceRoot string
	cursor        int
	editing       bool
	input         textinput.Model
	width, height int
	message       string
}

func NewSettingsPage(cfg *config.Config, workspaceRoot string) *SettingsPage {
	ti := textinput.New()
	ti.CharLimit = 128
	return &SettingsPage{
		cfg:           cfg,
		workspaceRoot: workspaceRoot,
		input:         ti,
	}
}

func (p *SettingsPage) Init() tea.Cmd { return nil }

func (p *SettingsPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.editing {
			switch msg.String() {
			case "enter":
				p.applyValue(p.input.Value())
				p.editing = false
				p.input.Blur()
				return p, nil
			case "esc":
				p.editing = false
				p.input.Blur()
				return p, nil
			}
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		}

		switch msg.String() {
		case "down":
			if p.cursor < len(settingFields)-1 {
				p.cursor++
			}
		case "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "enter", "e":
			p.editing = true
			p.input.SetValue(p.getValue(p.cursor))
			p.input.Focus()
			return p, p.input.Focus()
		case "s":
			if err := config.Save(*p.cfg, p.workspaceRoot, false); err != nil {
				p.message = fmt.Sprintf("Error saving: %v", err)
			} else {
				p.message = "Settings saved to workspace"
			}
		}
	}
	return p, nil
}

func (p *SettingsPage) View() string {
	var inner strings.Builder

	for i, f := range settingFields {
		cursor := "  "
		if i == p.cursor {
			cursor = ui.BoldStyle.Render("> ")
		}

		val := p.getValue(i)
		if val == "" {
			val = ui.DimStyle.Render("(not set)")
		}

		line := fmt.Sprintf("%s%-20s %s", cursor, f.label, val)
		inner.WriteString(line)
		inner.WriteString("\n")
	}

	if p.editing {
		inner.WriteString("\n")
		inner.WriteString(fmt.Sprintf("  Edit %s:\n", settingFields[p.cursor].label))
		inner.WriteString("  " + p.input.View())
		inner.WriteString("\n")
	}

	if p.message != "" {
		inner.WriteString("\n  " + p.message)
	}

	return ui.Panel("Settings", inner.String(), p.width, 0, false)
}

func (p *SettingsPage) Name() string { return "Settings" }

func (p *SettingsPage) ShortHelp() []key.Binding {
	if p.editing {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save to disk")),
	}
}

func (p *SettingsPage) InputCaptured() bool {
	return p.editing
}

func (p *SettingsPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *SettingsPage) getValue(idx int) string {
	switch settingFields[idx].key {
	case "default_board":
		return p.cfg.DefaultBoard
	case "serial_port":
		return p.cfg.SerialPort
	case "serial_baud_rate":
		return strconv.Itoa(p.cfg.SerialBaudRate)
	case "build_dir":
		return p.cfg.BuildDir
	case "flash_runner":
		return p.cfg.FlashRunner
	}
	return ""
}

func (p *SettingsPage) applyValue(val string) {
	switch settingFields[p.cursor].key {
	case "default_board":
		p.cfg.DefaultBoard = val
	case "serial_port":
		p.cfg.SerialPort = val
	case "serial_baud_rate":
		if n, err := strconv.Atoi(val); err == nil {
			p.cfg.SerialBaudRate = n
		}
	case "build_dir":
		p.cfg.BuildDir = val
	case "flash_runner":
		p.cfg.FlashRunner = val
	}
	p.message = fmt.Sprintf("%s updated", settingFields[p.cursor].label)
}
