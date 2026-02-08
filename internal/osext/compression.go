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
package osext

import (
	"compress/gzip"
	"io"
	"os"
)

const tempMask = "osext-*"

// UnGZIP decompresses a gzip file and returns a temporary file handler.
// it must be removed after use.  It expects r to contain a gzip file data.
func UnGZIP(r io.Reader) (*os.File, error) {
	return UnGZIPTo(r, "", tempMask)
}

func UnGZIPTo(r io.Reader, dir, mask string) (*os.File, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	f, err := os.CreateTemp("", tempMask)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, gr)
	if err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	// reset temporary file position to prepare it for reading.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return f, nil
}
