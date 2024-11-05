package menu

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type Style struct {
	Focused StyleSet
	Blurred StyleSet
}

type StyleSet struct {
	Border       lipgloss.Style
	Title        lipgloss.Style
	Description  lipgloss.Style
	Cursor       lipgloss.Style
	Item         lipgloss.Style
	ItemSelected lipgloss.Style
	ItemDisabled lipgloss.Style
}

func DefaultStyle() *Style {
	t := ui.DefaultTheme()
	return &Style{
		Focused: StyleSet{
			Border:       t.Focused.Border,
			Title:        t.Focused.Title,
			Description:  t.Focused.Description,
			Cursor:       t.Focused.Cursor,
			Item:         t.Focused.Text,
			ItemSelected: t.Focused.SelectedLine,
			ItemDisabled: t.Blurred.Text,
		},
		Blurred: StyleSet{
			Border:       t.Blurred.Border,
			Title:        t.Blurred.Title,
			Description:  t.Blurred.Description,
			Cursor:       t.Blurred.Cursor,
			Item:         t.Blurred.Text,
			ItemSelected: t.Blurred.SelectedLine,
			ItemDisabled: t.Blurred.Text,
		},
	}
}
