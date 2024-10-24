package cfgui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// Show initialises and runs the configuration UI.
func Show(ctx context.Context) error {
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
}

func New() configmodel {
	cfg := effectiveConfig()
	end := 0
	for _, group := range cfg {
		end += len(group.params)
	}
	end--
	return configmodel{
		cfg: effectiveConfig(),
		end: end,
	}
}
