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
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace/wspcfg"
	"github.com/rusq/slackdump/v4/internal/cache"
)

type EZLogin struct {
	Flags   map[string]bool `json:"flags"`
	Browser string          `json:"browser"`
}

func (inf *EZLogin) collect(PathReplFunc) {
	inf.Flags = cache.EzLoginFlags()
	inf.Browser = wspcfg.Browser.String()
}
