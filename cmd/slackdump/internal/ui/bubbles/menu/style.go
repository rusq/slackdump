// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
