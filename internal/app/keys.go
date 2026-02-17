package app

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	ToggleFocus key.Binding
	Help        key.Binding
	Quit        key.Binding
}

var GlobalKeys = KeyMap{
	ToggleFocus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle focus"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
