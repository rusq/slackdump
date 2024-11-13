package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

var DefaultHuhKeymap = huh.NewDefaultKeyMap()

func init() {
	// redefinition of some of the default keys.
	DefaultHuhKeymap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("esc", "quit"))
}
