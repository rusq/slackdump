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

package apiconfig

import (
	"testing"
	"testing/fstest"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/filemgr"
)

func Test_checkerModel_windowResize(t *testing.T) {
	f := filemgr.New(fstest.MapFS{}, ".", ".", 15, ConfigExts...)
	f.Width = filemgr.MinWidth
	m := checkerModel{
		files: f,
		view:  viewport.New(80-f.Width, f.Height),
	}

	const termWidth = 120
	updated, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: 40})
	got := updated.(checkerModel)

	if got.files.Width != filemgr.MinWidth {
		t.Errorf("files.Width = %d, want %d (file pane must stay fixed)", got.files.Width, filemgr.MinWidth)
	}
	if want := termWidth - filemgr.MinWidth; got.view.Width != want {
		t.Errorf("view.Width = %d, want %d (viewport must take the remaining width)", got.view.Width, want)
	}
}
