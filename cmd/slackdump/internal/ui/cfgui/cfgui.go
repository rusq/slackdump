package cfgui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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
		Style: Style{
			Border:        ui.DefaultTheme().Focused.Border,
			Title:         ui.DefaultTheme().Focused.Options.Section,
			Description:   ui.DefaultTheme().Focused.Description,
			Name:          ui.DefaultTheme().Focused.Options.Name,
			ValueEnabled:  ui.DefaultTheme().Focused.Options.EnabledValue,
			ValueDisabled: ui.DefaultTheme().Focused.Options.DisabledValue,
			SelectedName:  ui.DefaultTheme().Focused.Options.SelectedName,
			Cursor:        ui.DefaultTheme().Focused.Cursor,
		},
	}
}
