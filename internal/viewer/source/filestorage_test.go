package source

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func Test_fstMattermost_File(t *testing.T) {
	dir := t.TempDir()
	// Create a file in the __uploads directory.
	uploads := filepath.Join(dir, mmuploads, "file_id1")
	err := os.MkdirAll(uploads, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(uploads, "filename.ext"), []byte("file contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(dir)
	sub, err := fs.Sub(fsys, mmuploads)
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
			r := &fstMattermost{
				fs: tt.fields.fs,
			}
			got, err := r.File(tt.args.id, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("fstMattermost.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fstMattermost.File() = %v, want %v", got, tt.want)
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &fstStandard{
				fs:  tt.fields.fs,
				idx: tt.fields.idx,
			}
			got, err := r.File(tt.args.id, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("fstStandard.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fstStandard.File() = %v, want %v", got, tt.want)
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
		f       fstNotFound
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fstNotFound{}
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
			r := &fstDump{
				fs:  tt.fields.fs,
				idx: tt.fields.idx,
			}
			got, err := r.File(tt.args.id, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("fstDump.File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("fstDump.File() = %v, want %v", got, tt.want)
			}
		})
	}
}