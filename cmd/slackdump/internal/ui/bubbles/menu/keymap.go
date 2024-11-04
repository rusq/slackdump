package menu

import "github.com/charmbracelet/bubbles/key"

type Keymap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Quit   key.Binding
}

func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	}
}

func (k *Keymap) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Quit}
}
