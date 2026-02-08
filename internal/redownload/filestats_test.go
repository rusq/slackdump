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
	"reflect"
	"testing"

	"github.com/dustin/go-humanize"
)

func TestFileStats_add(t *testing.T) {
	fs := FileStats{NumFiles: 1, NumBytes: 10}
	fs.add(FileStats{NumFiles: 2, NumBytes: 20})

	if fs.NumFiles != 3 {
		t.Fatalf("NumFiles = %d, want %d", fs.NumFiles, 3)
	}
	if fs.NumBytes != 30 {
		t.Fatalf("NumBytes = %d, want %d", fs.NumBytes, 30)
	}
}

func TestFileStats_Attr(t *testing.T) {
	fs := FileStats{NumFiles: 5, NumBytes: 12345}

	attr := fs.Attr()

	if attr.Key != "file_stats" {
		t.Fatalf("attr.Key = %q, want %q", attr.Key, "file_stats")
	}
	if attr.Value.Kind() != slog.KindGroup {
		t.Fatalf("attr.Kind = %v, want %v", attr.Value.Kind(), slog.KindGroup)
	}

	wantGroup := []slog.Attr{
		slog.Uint64("num_files", uint64(fs.NumFiles)),
		slog.String("total_bytes", humanize.Bytes(fs.NumBytes)),
	}
	if gotGroup := attr.Value.Group(); !reflect.DeepEqual(gotGroup, wantGroup) {
		t.Fatalf("attr.Value.Group() = %#v, want %#v", gotGroup, wantGroup)
	}
}
