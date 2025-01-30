package chunk

import (
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"
)

func Test_version(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantID  FileID
		want    int64
		wantErr bool
	}{
		{
			name:    "test",
			args:    args{name: "channels.json.gz"},
			wantID:  FChannels,
			want:    0,
			wantErr: false,
		},
		{
			name:    "some version",
			args:    args{name: "channels_123.json.gz"},
			wantID:  FChannels,
			want:    123,
			wantErr: false,
		},
		{
			name:    "parse error",
			args:    args{name: "channels_abc.json.gz"},
			wantID:  "",
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, got, err := version(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.version() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Directory.version() = %v, want %v", got, tt.want)
			}
			if gotID != tt.wantID {
				t.Errorf("Directory.version() = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func Test_versions(t *testing.T) {
	type args struct {
		names []string
	}
	tests := []struct {
		name    string
		args    args
		want    []int64
		wantErr bool
	}{
		{
			name: "single file",
			args: args{
				names: []string{"channels.json.gz"},
			},
			want:    []int64{0},
			wantErr: false,
		},
		{
			name: "multiple files",
			args: args{
				names: []string{"channels_123.json.gz", "channels.json.gz", "channels_456.json.gz"},
			},
			want:    []int64{456, 123, 0},
			wantErr: false,
		},
		{
			name: "parse error",
			args: args{
				names: []string{"channels.json.gz", "channels_abc.json.gz", "channels_456.json.gz"},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := versions(tt.args.names...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.versions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Directory.versions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_collectVersions(t *testing.T) {
	type args struct {
		fsys fs.FS
	}
	tests := []struct {
		name    string
		args    args
		want    []fileversions
		wantErr bool
	}{
		{
			name: "returns proper versions",
			args: args{
				fsys: fstest.MapFS{
					"channels_123.json.gz": &fstest.MapFile{},
					"channels_124.json.gz": &fstest.MapFile{},
					"channels.json.gz":     &fstest.MapFile{},
					"C123451.json.gz":      &fstest.MapFile{},
					"C123451_123.json.gz":  &fstest.MapFile{},
				},
			},
			want: []fileversions{
				{ID: "C123451", V: []int64{123, 0}},
				{ID: FChannels, V: []int64{124, 123, 0}},
			},
			wantErr: false,
		},
		{
			name: "invalid file",
			args: args{
				fsys: fstest.MapFS{
					"channels_123.json.gz": &fstest.MapFile{},
					"channels_abc.json.gz": &fstest.MapFile{},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectVersions(tt.args.fsys)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.collectVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Directory.collectVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}
