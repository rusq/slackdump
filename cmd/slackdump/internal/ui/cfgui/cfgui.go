package cfgui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

// Global initialises and runs the configuration UI.
func Global(ctx context.Context) error {
	p := tea.NewProgram(NewConfigUI(DefaultStyle(), globalConfig))
	_, err := p.Run()
	return err
}

func Local(ctx context.Context, cfgFn func() Configuration) error {
	p := tea.NewProgram(NewConfigUI(DefaultStyle(), cfgFn))
	_, err := p.Run()
	return err
}

func NewConfigUI(sty Style, cfgFn func() Configuration) tea.Model {
	end := 0
	for _, group := range cfgFn() {
		end += len(group.Params)
	}
	end--
	return configmodel{
		cfgFn: cfgFn,
		end:   end,
		Style: sty,
	}
}

func DefaultStyle() Style {
	return Style{
		Border:        ui.DefaultTheme().Focused.Border,
		Title:         ui.DefaultTheme().Focused.Options.Section,
		Description:   ui.DefaultTheme().Focused.Description,
		Name:          ui.DefaultTheme().Focused.Options.Name,
		ValueEnabled:  ui.DefaultTheme().Focused.Options.EnabledValue,
		ValueDisabled: ui.DefaultTheme().Focused.Options.DisabledValue,
		SelectedName:  ui.DefaultTheme().Focused.Options.SelectedName,
		Cursor:        ui.DefaultTheme().Focused.Cursor,
	}
}
