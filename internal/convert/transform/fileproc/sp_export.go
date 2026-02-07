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
package fileproc

import (
	"github.com/rusq/slackdump/v3/processor"
	"github.com/rusq/slackdump/v3/source"
)

// NewExport initialises an export file subprocessor based on the given export
// type.  This subprocessor can be later plugged into the
// [expproc.Conversations] processor.
func NewExport(typ source.StorageType, dl Downloader) processor.Filer {
	switch typ {
	case source.STstandard:
		return NewWithPathFn(dl, source.StdFilepath)
	case source.STmattermost:
		return NewWithPathFn(dl, source.MattermostFilepath)
	default:
		return &processor.NopFiler{}
	}
}

// New creates a new file processor that uses mattermost file naming
// pattern.
func New(dl Downloader) processor.Filer {
	return NewWithPathFn(dl, source.MattermostFilepath)
}
