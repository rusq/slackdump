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

package viewer

import (
	"fmt"
	"io/fs"

	"github.com/rusq/slackdump/v4/source"
)

// fileByIDStorage is an optional extension of [source.Storage] that allows
// looking up a file by its ID alone, without knowing the filename.  All
// built-in storage types implement this interface; third-party implementations
// that do not will cause canvas features to degrade gracefully (tab shown as
// disabled) rather than error.
type fileByIDStorage interface {
	FileByID(id string) (string, error)
}

// fileByID looks up a file by ID alone via the optional [fileByIDStorage]
// extension interface.  If the concrete storage type does not implement it,
// the returned error wraps [fs.ErrNotExist] so all callers can use
// [errors.Is] for graceful degradation.
func fileByID(storage source.Storage, id string) (string, error) {
	fb, ok := storage.(fileByIDStorage)
	if !ok {
		return "", fmt.Errorf("storage type %T does not implement FileByID: %w", storage, fs.ErrNotExist)
	}
	return fb.FileByID(id)
}
