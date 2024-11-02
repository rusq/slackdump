package cfgui

import "github.com/charmbracelet/bubbles/key"

type Keymap struct {
	Up      key.Binding
	Down    key.Binding
	Home    key.Binding
	End     key.Binding
	Refresh key.Binding
	Select  key.Binding
	Quit    key.Binding
}

func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
		Home:    key.NewBinding(key.WithKeys("home"), key.WithHelp("home/end", "top/bottom")),
		End:     key.NewBinding(key.WithKeys("end")),
		Refresh: key.NewBinding(key.WithKeys("f5", "ctrl+r"), key.WithHelp("f5", "refresh")),
		Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Quit:    key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k *Keymap) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Home, k.Refresh, k.Select, k.Quit}
}
