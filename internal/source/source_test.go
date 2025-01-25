// Package source provides archive readers for different output formats.
//
// Currently, the following formats are supported:
//   - archive
//   - Slack Export
//   - dump
package source

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
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
