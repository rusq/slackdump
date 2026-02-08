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
package directory

import "github.com/rusq/slackdump/v3/internal/chunk"

// Workspace is a processor that writes the workspace information into the
// workspace file.
type Workspace struct {
	*dirproc
}

// NewWorkspace creates a new workspace processor.
func NewWorkspace(cd *chunk.Directory) (*Workspace, error) {
	p, err := newDirProc(cd, chunk.FWorkspace)
	if err != nil {
		return nil, err
	}
	return &Workspace{dirproc: p}, nil
}
