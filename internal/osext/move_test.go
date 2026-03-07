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
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"
	fx "github.com/rusq/slackdump/v4/internal/fixtures"
)

func TestMoveFile(t *testing.T) {
	d := t.TempDir()

	// fixtures

	fsa := fsadapter.NewDirectory(d)
	defer fsa.Close()

	// create source file
	srcf := filepath.Join(d, "src")
	fx.MkTestFileName(t, srcf, "test")

	type args struct {
		src string
		fs  fsadapter.FS
		dst string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"existing source file",
			args{srcf, fsa, "dst"},
			false,
		},
		{
			"non-existing source file",
			args{filepath.Join(d, "non-existing"), fsa, "dst"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MoveFile(tt.args.src, tt.args.fs, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("MoveFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
