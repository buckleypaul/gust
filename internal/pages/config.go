package pages

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/buckleypaul/gust/internal/app"
	"github.com/buckleypaul/gust/internal/ui"
)

// kconfigEntry represents a single Kconfig symbol from prj.conf.
type kconfigEntry struct {
	Name    string
	Value   string
	Comment string // inline comment
}

type ConfigPage struct {
	workspaceRoot string
	entries       []kconfigEntry
	filtered      []kconfigEntry
	cursor        int
	search        textinput.Model
	searching     bool
	width, height int
	message       string
	loaded        bool
}

func NewConfigPage(workspaceRoot string) *ConfigPage {
	ti := textinput.New()
	ti.Placeholder = "Search symbols..."
	ti.CharLimit = 64
	return &ConfigPage{
		workspaceRoot: workspaceRoot,
		search:        ti,
	}
}

func (p *ConfigPage) Init() tea.Cmd {
	return p.loadConfig
}

func (p *ConfigPage) Update(msg tea.Msg) (app.Page, tea.Cmd) {
	switch msg := msg.(type) {
	case kconfigLoadedMsg:
		p.loaded = true
		if msg.err != nil {
			p.message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			p.entries = msg.entries
			p.filterEntries()
		}
		return p, nil

	case tea.KeyMsg:
		if p.searching {
			switch msg.String() {
			case "enter", "esc":
				p.searching = false
				p.search.Blur()
				return p, nil
			}
			var cmd tea.Cmd
			p.search, cmd = p.search.Update(msg)
			p.filterEntries()
			return p, cmd
		}

		switch msg.String() {
		case "/":
			p.searching = true
			p.search.Focus()
			return p, p.search.Focus()
		case "j", "down":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "r":
			return p, p.loadConfig
		}
	}
	return p, nil
}

func (p *ConfigPage) View() string {
	var b strings.Builder
	b.WriteString(ui.Title("Config"))
	b.WriteString("\n")

	if p.message != "" {
		b.WriteString("  " + p.message + "\n\n")
	}

	if !p.loaded {
		b.WriteString("  Loading prj.conf...")
		return b.String()
	}

	if p.searching {
		b.WriteString("  " + p.search.View() + "\n\n")
	}

	if len(p.filtered) == 0 {
		b.WriteString(ui.DimStyle.Render("  No Kconfig symbols found."))
		return b.String()
	}

	listHeight := p.height - 6
	if listHeight < 5 {
		listHeight = 5
	}

	start := p.cursor - listHeight/2
	if start < 0 {
		start = 0
	}
	end := start + listHeight
	if end > len(p.filtered) {
		end = len(p.filtered)
		start = end - listHeight
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		e := p.filtered[i]
		cursor := "  "
		if i == p.cursor {
			cursor = ui.BoldStyle.Render("> ")
		}
		line := fmt.Sprintf("%s%-40s = %s", cursor, e.Name, e.Value)
		if e.Comment != "" {
			line += ui.DimStyle.Render("  # " + e.Comment)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString(fmt.Sprintf("\n  %d/%d symbols", p.cursor+1, len(p.filtered)))
	if p.search.Value() != "" {
		b.WriteString(fmt.Sprintf(" (filter: %s)", p.search.Value()))
	}
	b.WriteString("\n")

	return b.String()
}

func (p *ConfigPage) Name() string { return "Config" }

func (p *ConfigPage) ShortHelp() []key.Binding {
	if p.searching {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "done")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	}
}

func (p *ConfigPage) SetSize(w, h int) {
	p.width = w
	p.height = h
}

type kconfigLoadedMsg struct {
	entries []kconfigEntry
	err     error
}

func (p *ConfigPage) loadConfig() tea.Msg {
	confPath := filepath.Join(p.workspaceRoot, "prj.conf")
	entries, err := parsePrjConf(confPath)
	return kconfigLoadedMsg{entries: entries, err: err}
}

func (p *ConfigPage) filterEntries() {
	query := strings.ToLower(p.search.Value())
	if query == "" {
		p.filtered = p.entries
	} else {
		p.filtered = nil
		for _, e := range p.entries {
			if strings.Contains(strings.ToLower(e.Name), query) ||
				strings.Contains(strings.ToLower(e.Value), query) {
				p.filtered = append(p.filtered, e)
			}
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func parsePrjConf(path string) ([]kconfigEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []kconfigEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse CONFIG_FOO=value # optional comment
		var comment string
		if idx := strings.Index(line, "#"); idx != -1 {
			comment = strings.TrimSpace(line[idx+1:])
			line = strings.TrimSpace(line[:idx])
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		entries = append(entries, kconfigEntry{
			Name:    strings.TrimSpace(parts[0]),
			Value:   strings.TrimSpace(parts[1]),
			Comment: comment,
		})
	}

	return entries, scanner.Err()
}
