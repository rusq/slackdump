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

package dumpui

import (
	"testing"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/cfgui"
)

func TestWizard_helpText(t *testing.T) {
	tests := []struct {
		name string
		w    Wizard
		want string
	}{
		{
			name: "explicit Help takes precedence",
			w:    Wizard{Help: "override", Cmd: &base.Command{Long: "long help"}},
			want: "override",
		},
		{
			name: "falls back to command long help",
			w:    Wizard{Cmd: &base.Command{Long: "long help"}},
			want: "long help",
		},
		{
			name: "empty when no help available",
			w:    Wizard{Cmd: &base.Command{}},
			want: "",
		},
		{
			name: "empty when no command",
			w:    Wizard{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.w.helpText(); got != tt.want {
				t.Errorf("helpText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWizard_items(t *testing.T) {
	itemIDs := func(w *Wizard) []string {
		var ids []string
		for _, it := range w.items() {
			if it.Separator {
				ids = append(ids, "---")
				continue
			}
			ids = append(ids, it.ID)
		}
		return ids
	}
	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	t.Run("full wizard has all items including help", func(t *testing.T) {
		w := &Wizard{
			Name:        "Dump",
			Cmd:         &base.Command{Long: "long help"},
			LocalConfig: func() cfgui.Configuration { return cfgui.Configuration{} },
		}
		want := []string{actLocalConfig, actRun, actHelp, "---", actGlobalConfig, "---", actExit}
		if got := itemIDs(w); !equal(got, want) {
			t.Errorf("items() IDs = %v, want %v", got, want)
		}
	})

	t.Run("no local config and no help", func(t *testing.T) {
		w := &Wizard{Name: "Dump", Cmd: &base.Command{}}
		want := []string{actRun, "---", actGlobalConfig, "---", actExit}
		if got := itemIDs(w); !equal(got, want) {
			t.Errorf("items() IDs = %v, want %v", got, want)
		}
	})
}
