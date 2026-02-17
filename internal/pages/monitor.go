package pages

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	serialpkg "github.com/buckleypaul/gust/internal/serial"
	"github.com/buckleypaul/gust/internal/store"
	"github.com/buckleypaul/gust/internal/ui"
)

type monitorState int

const (
	monitorStatePortSelect monitorState = iota
	monitorStateConnected
)

// serialDataMsg wraps data received from the serial port.
type serialDataMsg struct {
	Data string
}

type MonitorPage struct {
	state      monitorState
	ports      []serialpkg.PortInfo
	cursor     int
	monitor    *serialpkg.Monitor
	output     strings.Builder
	viewport   viewport.Model
	input      textinput.Model
	autoScroll bool
	store      *store.Store
	baudRate   int
	width, height int
	message    string
	program    *tea.Program
}

func NewMonitorPage(s *store.Store, baudRate int) *MonitorPage {
	vp := viewport.New(0, 0)
	ti := textinput.New()
	ti.Placeholder = "Type to send..."
	ti.CharLimit = 256

	if baudRate == 0 {
		baudRate = 115200
	}

	return &MonitorPage{
		monitor:    serialpkg.NewMonitor(),
		viewport:   vp,
		input:      ti,
		autoScroll: true,
		store:      s,
		baudRate:   baudRate,
	}
}

func (p *MonitorPage) Init() tea.Cmd {
	return p.refreshPorts
}

func (p *MonitorPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case portsLoadedMsg:
		p.ports = msg.ports
		if msg.err != nil {
			p.message = fmt.Sprintf("Error listing ports: %v", msg.err)
		}
		return p, nil

	case serialDataMsg:
		p.output.WriteString(msg.Data)
		p.viewport.SetContent(p.output.String())
		if p.autoScroll {
			p.viewport.GotoBottom()
		}
		return p, p.waitForData

	case tea.KeyMsg:
		switch p.state {
		case monitorStatePortSelect:
			switch msg.String() {
			case "down":
				if p.cursor < len(p.ports)-1 {
					p.cursor++
				}
			case "up":
				if p.cursor > 0 {
					p.cursor--
				}
			case "r":
				return p, p.refreshPorts
			case "enter":
				if len(p.ports) > 0 {
					return p, p.connect(p.ports[p.cursor].Name)
				}
			}

		case monitorStateConnected:
			switch msg.String() {
			case "d":
				p.monitor.Disconnect()
				p.state = monitorStatePortSelect
				p.message = "Disconnected"
				return p, nil
			case "s":
				p.autoScroll = !p.autoScroll
				return p, nil
			case "c":
				p.output.Reset()
				p.viewport.SetContent("")
				return p, nil
			case "enter":
				if p.input.Value() != "" {
					data := p.input.Value() + "\r\n"
					p.monitor.Write([]byte(data))
					p.input.SetValue("")
				}
				return p, nil
			}
			// Forward to input
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		}
	}

	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p *MonitorPage) View() string {
	var b strings.Builder

	switch p.state {
	case monitorStatePortSelect:
		var connB strings.Builder
		if p.message != "" {
			connB.WriteString("  " + p.message + "\n\n")
		}
		if len(p.ports) == 0 {
			connB.WriteString(ui.DimStyle.Render("  No serial ports found. Press r to refresh."))
			connB.WriteString("\n")
		} else {
			for i, port := range p.ports {
				cursor := "  "
				if i == p.cursor {
					cursor = ui.BoldStyle.Render("> ")
				}
				desc := port.Name
				if port.IsUSB {
					desc += fmt.Sprintf(" (USB %s:%s)", port.VID, port.PID)
				}
				connB.WriteString(cursor + desc + "\n")
			}
			connB.WriteString(fmt.Sprintf("\n  Baud rate: %d\n", p.baudRate))
		}
		b.WriteString(ui.Panel("Connection", connB.String(), p.width, 0, false))

	case monitorStateConnected:
		var connB strings.Builder
		if p.message != "" {
			connB.WriteString("  " + p.message + "\n")
		}
		scrollStatus := "ON"
		if !p.autoScroll {
			scrollStatus = "OFF"
		}
		connB.WriteString(fmt.Sprintf("  Auto-scroll: %s\n", scrollStatus))
		b.WriteString(ui.Panel("Connection", connB.String(), p.width, 0, false))
		b.WriteString("\n")

		var outB strings.Builder
		outB.WriteString(p.viewport.View())
		outB.WriteString("\n")
		outB.WriteString(p.input.View())
		b.WriteString(ui.Panel("Output", outB.String(), p.width, 0, false))
	}

	return b.String()
}

func (p *MonitorPage) Name() string { return "Monitor" }

func (p *MonitorPage) ShortHelp() []key.Binding {
	if p.state == monitorStateConnected {
		return []key.Binding{
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "disconnect")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "auto-scroll")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

func (p *MonitorPage) InputCaptured() bool {
	return p.state == monitorStateConnected
}

func (p *MonitorPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	vpHeight := h - 8
	if vpHeight < 3 {
		vpHeight = 3
	}
	p.viewport.Width = w - 4
	p.viewport.Height = vpHeight
}

type portsLoadedMsg struct {
	ports []serialpkg.PortInfo
	err   error
}

func (p *MonitorPage) refreshPorts() tea.Msg {
	ports, err := serialpkg.ListPorts()
	return portsLoadedMsg{ports: ports, err: err}
}

func (p *MonitorPage) connect(portName string) tea.Cmd {
	return func() tea.Msg {
		err := p.monitor.Connect(portName, p.baudRate)
		if err != nil {
			return portsLoadedMsg{err: err}
		}
		p.state = monitorStateConnected
		p.message = fmt.Sprintf("Connected to %s @ %d", portName, p.baudRate)
		p.input.Focus()
		// Start reading data
		return p.waitForDataMsg()
	}
}

func (p *MonitorPage) waitForData() tea.Msg {
	return p.waitForDataMsg()
}

func (p *MonitorPage) waitForDataMsg() tea.Msg {
	if !p.monitor.Connected() {
		return nil
	}
	data, ok := <-p.monitor.DataChan()
	if !ok {
		return nil
	}
	return serialDataMsg{Data: data}
}
