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
