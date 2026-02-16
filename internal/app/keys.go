package app

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	NextPage key.Binding
	PrevPage key.Binding
	Page1    key.Binding
	Page2    key.Binding
	Page3    key.Binding
	Page4    key.Binding
	Page5    key.Binding
	Page6    key.Binding
	Page7    key.Binding
	Page8    key.Binding
	Page9    key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var GlobalKeys = KeyMap{
	NextPage: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next page"),
	),
	PrevPage: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev page"),
	),
	Page1: key.NewBinding(key.WithKeys("1")),
	Page2: key.NewBinding(key.WithKeys("2")),
	Page3: key.NewBinding(key.WithKeys("3")),
	Page4: key.NewBinding(key.WithKeys("4")),
	Page5: key.NewBinding(key.WithKeys("5")),
	Page6: key.NewBinding(key.WithKeys("6")),
	Page7: key.NewBinding(key.WithKeys("7")),
	Page8: key.NewBinding(key.WithKeys("8")),
	Page9: key.NewBinding(key.WithKeys("9")),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
