package cfgui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type Style struct {
	Focused StyleSet
	Blurred StyleSet
}

type StyleSet struct {
	Border        lipgloss.Style
	Title         lipgloss.Style
	Description   lipgloss.Style
	Name          lipgloss.Style
	ValueEnabled  lipgloss.Style
	ValueDisabled lipgloss.Style
	SelectedName  lipgloss.Style
	Cursor        lipgloss.Style
}

func DefaultStyle() *Style {
	t := ui.DefaultTheme()
	return &Style{
		Focused: StyleSet{
			Border:        t.Focused.Border,
			Title:         t.Focused.Options.Section,
			Description:   t.Focused.Description,
			Name:          t.Focused.Options.Name,
			ValueEnabled:  t.Focused.Options.EnabledValue,
			ValueDisabled: t.Focused.Options.DisabledValue,
			SelectedName:  t.Focused.Options.SelectedName,
			Cursor:        t.Focused.Cursor,
		},
		Blurred: StyleSet{
			Border:        t.Blurred.Border,
			Title:         t.Blurred.Options.Section,
			Description:   t.Blurred.Description,
			Name:          t.Blurred.Options.Name,
			ValueEnabled:  t.Blurred.Options.EnabledValue,
			ValueDisabled: t.Blurred.Options.DisabledValue,
			SelectedName:  t.Blurred.Options.SelectedName,
			Cursor:        t.Blurred.Cursor,
		},
	}
}
