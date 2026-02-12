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

package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func Test_fstMattermost_File(t *testing.T) {
	dir := t.TempDir()
	// Create a file in the __uploads directory.
	uploads := filepath.Join(dir, chunk.UploadsDir, "file_id1")
	err := os.MkdirAll(uploads, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(uploads, "filename.ext"), []byte("file contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(dir)
	sub, err := fs.Sub(fsys, chunk.UploadsDir)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		fs fs.FS
	}
	type args struct {
		id   string
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "file exists",
			fields: fields{
				fs: sub,
			},
			args: args{
				id:   "file_id1",
				name: "filename.ext",
			},
			want:    "file_id1/filename.ext",
			wantErr: false,
		},
		{
			name: "file does not exist",
			fields: fields{
				fs: sub,
			},
			args: args{
				id:   "file_id1",
				name: "nonexistent.ext",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &STMattermost{
				fs: tt.fields.fs,
			}
			got, err := r.File(tt.args.id, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("STMattermost.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("STMattermost.File() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fstStandard_File(t *testing.T) {
	type fields struct {
		fs  fs.FS
		idx map[string]string
	}
	type args struct {
		id  string
		in1 string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "file is present in index, and in filesystem",
			fields: fields{
				fs: fstest.MapFS{
					"C123/attachments/F123-test.txt": {
						Data: []byte("test file"),
						Mode: 644,
					}},
				idx: map[string]string{
					"F123": "C123/attachments/F123-test.txt",
				},
			},
			args: args{
				id:  "F123",
				in1: "",
			},
			want:    "C123/attachments/F123-test.txt",
			wantErr: false,
		},
		{
			name: "file is present in index, but not on the filesystem",
			fields: fields{
				fs: fstest.MapFS{},
				idx: map[string]string{
					"F123": "C123/attachments/F123-test.txt",
				},
			},
			args: args{
				id:  "F123",
				in1: "",
			},
			wantErr: true,
		},
		{
			name: "file is not in index",
			fields: fields{
				fs: fstest.MapFS{
					"C123/attachments/F123-test.txt": {
						Data: []byte("test file"),
						Mode: 644,
					}},
				idx: map[string]string{},
			},
			args: args{
				id:  "F123",
				in1: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &STStandard{
				fs:  tt.fields.fs,
				idx: tt.fields.idx,
			}
			got, err := r.File(tt.args.id, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("STStandard.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("STStandard.File() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fstNotFound_File(t *testing.T) {
	type args struct {
		id   string
		name string
	}
	tests := []struct {
		name    string
		f       NoStorage
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NoStorage{}
			got, err := f.File(tt.args.id, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("fstNotFound.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fstNotFound.File() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fstDump_File(t *testing.T) {
	type fields struct {
		fs  fs.FS
		idx map[string]string
	}
	type args struct {
		id   string
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &STDump{
				fs:  tt.fields.fs,
				idx: tt.fields.idx,
			}
			got, err := r.File(tt.args.id, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("STDump.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("STDump.File() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic replacements
		{"no change", "file.txt", "file.txt"},
		{"slashes", "foo/bar.txt", "foo_bar.txt"},
		{"backslashes", "foo\\bar.txt", "foo_bar.txt"},
		{"colons", "foo:bar.txt", "foo_bar.txt"},
		{"asterisk", "foo*bar.txt", "foo_bar.txt"},
		{"question mark", "foo?bar.txt", "foo_bar.txt"},
		{"quotes", "foo\"bar.txt", "foo_bar.txt"},
		{"less than", "foo<bar.txt", "foo_bar.txt"},
		{"greater than", "foo>bar.txt", "foo_bar.txt"},
		{"pipe", "foo|bar.txt", "foo_bar.txt"},
		// Trailing spaces and periods
		{"trailing space", "foo.txt ", "foo.txt"},
		{"trailing period", "foo.txt.", "foo.txt"},
		{"trailing space and period", "foo.txt .", "foo.txt"},
		{"multiple trailing space", "foo.txt    ", "foo.txt"},
		// Reserved names
		{"reserved CON", "CON", "_CON"},
		{"reserved PRN", "PRN.txt", "_PRN.txt"},
		{"reserved LPT1", "LPT1", "_LPT1"},
		{"reserved LPT9.txt", "LPT9.txt", "_LPT9.txt"},
		{"reserved com1", "com1", "_com1"},
		// Empty after sanitization
		{"empty string", "", "unnamed_file"},
		{"all invalid", "<>:\"/\\|?*", "_________"},
		// Unicode and safe
		{"unicode safe", "файл.txt", "файл.txt"},
		{"emoji", "file😀.txt", "file😀.txt"},
		// Dots in the middle
		{"dots in middle", "foo.bar.baz.txt", "foo.bar.baz.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			got := SanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
			}
			// Check if the sanitized filename can be created in the temp directory
			filePath := filepath.Join(dir, got)
			err := os.WriteFile(filePath, []byte("test content"), 0o644)
			if err != nil {
				t.Errorf("Failed to create file %q: %v", filePath, err)
			}
		})
	}
}
