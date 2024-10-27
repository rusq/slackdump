package cfgui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// Global initialises and runs the configuration UI.
func Global(ctx context.Context) error {
	m := NewConfigUI(DefaultStyle(), globalConfig)
	m.SetFocus(true)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func GlobalConfig() Configuration {
	return globalConfig()
}

func Local(ctx context.Context, cfgFn func() Configuration) error {
	p := tea.NewProgram(NewConfigUI(DefaultStyle(), cfgFn))
	_, err := p.Run()
	return err
}
