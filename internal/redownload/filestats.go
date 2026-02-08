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
package redownload

import (
	"log/slog"

	"github.com/dustin/go-humanize"
)

// FileStats contains the file statistics.
type FileStats struct {
	NumFiles uint
	NumBytes uint64
}

func (fs *FileStats) add(other FileStats) {
	fs.NumFiles += other.NumFiles
	fs.NumBytes += other.NumBytes
}

func (fs *FileStats) Attr() slog.Attr {
	return slog.Group("file_stats", slog.Uint64("num_files", uint64(fs.NumFiles)), slog.String("total_bytes", humanize.Bytes(fs.NumBytes)))
}
