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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	fx "github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestUnGZIP(t *testing.T) {
	d := t.TempDir()

	// fixtures

	//  uncompressed file
	uncompressed := filepath.Join(d, "file")
	fx.MkTestFileName(t, uncompressed, "test")
	uncF, err := os.Open(uncompressed)
	if err != nil {
		t.Fatal(err)
	}
	defer uncF.Close()

	// compressed file
	compressed := filepath.Join(d, "file.gz")
	{
		f, err := os.Create(compressed)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		gw := gzip.NewWriter(f)
		if _, err := gw.Write([]byte("test\n")); err != nil {
			t.Fatal(err)
		}
		if err := gw.Close(); err != nil {
			t.Fatal(err)
		}

		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}

	compF, err := os.Open(compressed)
	if err != nil {
		t.Fatal(err)
	}
	defer compF.Close()

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name        string
		args        args
		wantContent []byte
		wantErr     bool
	}{
		{
			"uncompressed",
			args{uncF},
			nil,
			true,
		},
		{
			"compressed",
			args{compF},
			[]byte("test\n"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnGZIP(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnGZIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Close()
			if tt.wantContent == nil {
				return
			}
			d, err := io.ReadAll(got)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantContent, d)
		})
	}
}
