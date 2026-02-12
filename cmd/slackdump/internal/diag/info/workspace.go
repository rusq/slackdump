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

package info

import (
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace"
)

type Workspace struct {
	Path       string `json:"path"`
	TxtExists  bool   `json:"txt_exists"`
	HasDefault bool   `json:"has_default"`
	Count      int    `json:"count"`
}

func (inf *Workspace) collect(replaceFn PathReplFunc) {
	inf.Path = replaceFn(cfg.CacheDir())
	inf.Count = -1
	// Workspace information
	m, err := workspace.CacheMgr()
	if err != nil {
		inf.Path = loser(err)
		return
	}
	if _, err := m.Current(); err == nil {
		inf.TxtExists = true
	}
	wsp, err := m.List()
	if err == nil {
		inf.Count = len(wsp)
	}
}
