package source

import (
	"context"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

var fixturesDir = filepath.Join("..", "fixtures", "assets")

func TestLoad(t *testing.T) {
	type args struct {
		ctx context.Context
		src string
	}
	tests := []struct {
		name    string
		args    args
		want    Sourcer
		wantErr bool
	}{
		{
			"chunk",
			args{context.Background(), filepath.Join(fixturesDir, "source_archive")},
			&ChunkDir{},
			false,
		},
		{
			"chunk, no workspace file",
			args{context.Background(), filepath.Join(fixturesDir, "source_archive_no_wsp")},
			&ChunkDir{},
			false,
		},
		{
			"export",
			args{context.Background(), filepath.Join(fixturesDir, "source_export.zip")},
			&Export{},
			false,
		},
		{
			"export dir",
			args{context.Background(), filepath.Join(fixturesDir, "source_export_dir")},
			&Export{},
			false,
		},
		{
			"dump.zip",
			args{context.Background(), filepath.Join(fixturesDir, "source_dump.zip")},
			&Dump{},
			false,
		},
		{
			"dump dir",
			args{context.Background(), filepath.Join(fixturesDir, "source_dump_dir")},
			&Dump{},
			false,
		},
		{
			"database directory",
			args{context.Background(), filepath.Join(fixturesDir, "source_database")},
			&Database{},
			false,
		},
		{
			"database file",
			args{context.Background(), filepath.Join(fixturesDir, "source_database.db")},
			&Database{},
			false,
		},
		{
			"unknown",
			args{context.Background(), filepath.Join(fixturesDir, "source_unknown")},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args.ctx, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			wantT := reflect.TypeOf(tt.want)
			gotT := reflect.TypeOf(got)
			if wantT != gotT {
				t.Errorf("Load() = %v, want %v", gotT, wantT)
			}
		})
	}
}

func TestFlags_String(t *testing.T) {
	tests := []struct {
		name string
		f    Flags
		want string
	}{
		{"unknown", 0, "unknown"},
		{"FChunk", FChunk, "chunk"},
		{"FDatabase|FMattermost", FDatabase | FDirectory, "..D....d"},
		{"all", FDatabase | FDump | FExport | FChunk | FZip | FDirectory, "..DUECzd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("Flags.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unmarshalOne(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}
	testfs := fstest.MapFS{
		"duke.json": {
			Data: []byte(`{"name":"duke nukem"}`),
		},
		"invalid_data.json": {
			Data: []byte(`{"name":42}`),
		},
	}

	type args struct {
		fsys fs.FS
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    testStruct
		wantErr bool
	}{
		{
			"duke",
			args{testfs, "duke.json"},
			testStruct{Name: "duke nukem"},
			false,
		},
		{
			"not found",
			args{testfs, "notfound.json"},
			testStruct{},
			true,
		},
		{
			"invalid data",
			args{testfs, "invalid_data.json"},
			testStruct{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalOne[testStruct](tt.args.fsys, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalOne() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshalOne() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unmarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}
	testSlice := []testStruct{
		{Name: "duke nukem"},
		{Name: "quake ranger"},
	}
	testfs := fstest.MapFS{
		"games.json": {
			Data: []byte(`[{"name":"duke nukem"},{"name":"quake ranger"}]`),
		},
		"invalid_data.json": {
			Data: []byte(`{"name":42}`),
		},
	}
	type args struct {
		fsys fs.FS
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    []testStruct
		wantErr bool
	}{
		{
			"games",
			args{testfs, "games.json"},
			testSlice,
			false,
		},
		{
			"not found",
			args{testfs, "notfound.json"},
			nil,
			true,
		},
		{
			"invalid data",
			args{testfs, "invalid_data.json"},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshal[[]testStruct](tt.args.fsys, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
