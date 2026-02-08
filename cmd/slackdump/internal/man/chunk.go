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
package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/chunk.md
var mdChunk string

var Chunk = &base.Command{
	UsageLine: "slackdump chunk",
	Short:     "chunk file format specification",
	Long:      mdChunk,
}
