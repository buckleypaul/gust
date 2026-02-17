package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/buckleypaul/gust/internal/ui"
)

// PickerItem represents a selectable item in the picker.
type PickerItem struct {
	Label string // Display text (project path)
	Value string // Selection value
	Desc  string // Optional secondary text (source label)
}

// PickerSelectedMsg is sent when the user selects an item.
type PickerSelectedMsg struct {
	Value string
}

// PickerClosedMsg is sent when the user closes the picker without selecting.
type PickerClosedMsg struct{}

// Picker is a filtered-list overlay component.
type Picker struct {
	title    string
	items    []PickerItem
	filtered []PickerItem
	input    textinput.Model
	cursor   int
	width    int
	height   int
}

const maxPickerItems = 12

// NewPicker creates a new picker overlay.
func NewPicker(title string) *Picker {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 128

	return &Picker{
		title: title,
		input: ti,
	}
}

// SetItems populates the picker with items.
func (p *Picker) SetItems(items []PickerItem) {
	p.items = items
	p.filter()
}

// SetSize sets the available dimensions.
func (p *Picker) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// Update handles input for the picker.
func (p *Picker) Update(msg tea.Msg) (*Picker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return p, func() tea.Msg { return PickerClosedMsg{} }
		case "enter":
			if len(p.filtered) > 0 && p.cursor < len(p.filtered) {
				value := p.filtered[p.cursor].Value
				return p, func() tea.Msg { return PickerSelectedMsg{Value: value} }
			}
			return p, nil
		case "up":
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil
		case "down":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
			return p, nil
		}
	}

	// Forward other keys to text input
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.filter()
	return p, cmd
}

// View renders the picker overlay.
func (p *Picker) View() string {
	boxWidth := p.width - 4
	if boxWidth > 60 {
		boxWidth = 60
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	innerWidth := boxWidth - 4 // border + padding

	var b strings.Builder

	// Input
	p.input.Width = innerWidth - 3 // account for prompt "> "
	b.WriteString(p.input.View())
	b.WriteString("\n\n")

	// Items
	visible := maxPickerItems
	if visible > len(p.filtered) {
		visible = len(p.filtered)
	}

	// Scroll window around cursor
	start := 0
	if p.cursor >= visible {
		start = p.cursor - visible + 1
	}
	end := start + visible
	if end > len(p.filtered) {
		end = len(p.filtered)
		start = end - visible
		if start < 0 {
			start = 0
		}
	}

	selectedStyle := lipgloss.NewStyle().Foreground(ui.Primary).Bold(true)

	for i := start; i < end; i++ {
		item := p.filtered[i]
		label := item.Label
		if len(label) > innerWidth-4 {
			label = label[:innerWidth-4]
		}

		if i == p.cursor {
			b.WriteString(selectedStyle.Render("> " + label))
		} else {
			b.WriteString("  " + label)
		}
		b.WriteString("\n")
	}

	if len(p.filtered) == 0 {
		b.WriteString(ui.DimStyle.Render("  No matches"))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	footer := fmt.Sprintf("(%d/%d projects)  esc:close", len(p.filtered), len(p.items))
	b.WriteString(ui.DimStyle.Render(footer))

	box := lipgloss.NewStyle().
		Width(boxWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Padding(1, 1).
		Render(b.String())

	// Add title to border
	titleStr := lipgloss.NewStyle().
		Foreground(ui.Primary).
		Bold(true).
		Render(" " + p.title + " ")

	// Replace the top border segment with the title
	lines := strings.Split(box, "\n")
	if len(lines) > 0 {
		topBorder := lines[0]
		// Insert title after the first 3 characters of the border
		if len(topBorder) > 4 {
			runes := []rune(topBorder)
			titleRunes := []rune(titleStr)
			insertPos := 3
			if insertPos+len(titleRunes) < len(runes) {
				result := make([]rune, 0, len(runes))
				result = append(result, runes[:insertPos]...)
				result = append(result, titleRunes...)
				result = append(result, runes[insertPos+len(titleRunes):]...)
				lines[0] = string(result)
			}
		}
		box = strings.Join(lines, "\n")
	}

	return box
}

func (p *Picker) filter() {
	query := strings.ToLower(p.input.Value())
	if query == "" {
		p.filtered = p.items
	} else {
		p.filtered = nil
		for _, item := range p.items {
			if fuzzyMatch(strings.ToLower(item.Label), query) {
				p.filtered = append(p.filtered, item)
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

// fuzzyMatch checks if all characters in query appear in s in order.
func fuzzyMatch(s, query string) bool {
	qi := 0
	for i := 0; i < len(s) && qi < len(query); i++ {
		if s[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}
