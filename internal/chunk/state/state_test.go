package state

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_tsUpdate(t *testing.T) {
	type args struct {
		m   map[string]int64
		id  string
		val string
	}
	tests := []struct {
		name    string
		args    args
		wantMap map[string]int64
	}{
		{
			"valid ts",
			args{
				map[string]int64{},
				"channel",
				"1638494510.037400",
			},
			map[string]int64{
				"channel": 1638494510037400,
			},
		},
		{
			"invalid ts",
			args{
				map[string]int64{},
				"channel",
				"x",
			},
			map[string]int64{},
		},
		{
			"newer ts",
			args{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel",
				"1638494510.037401",
			},
			map[string]int64{
				"channel": 1638494510037401,
			},
		},
		{
			"older ts",
			args{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel",
				"1638494510.037399",
			},
			map[string]int64{
				"channel": 1638494510037400,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsUpdate(tt.args.m, tt.args.id, tt.args.val)
			if !reflect.DeepEqual(tt.args.m, tt.wantMap) {
				t.Errorf("tsUpdate() = %v, want %v", tt.args.m, tt.wantMap)
			}
		})
	}
}

func Test_has(t *testing.T) {
	type args[T any] struct {
		m  map[string]T
		id string
	}
	tests := []struct {
		name string
		args args[int64]
		want bool
	}{
		{
			"empty map",
			args[int64]{
				map[string]int64{},
				"channel",
			},
			false,
		},
		{
			"not found",
			args[int64]{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel2",
			},
			false,
		},
		{
			"found",
			args[int64]{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel",
			},
			true,
		},
		{
			"nil map",
			args[int64]{
				nil,
				"channel",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := has(tt.args.m, tt.args.id); got != tt.want {
				t.Errorf("has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_latest(t *testing.T) {
	type args struct {
		m  map[string]int64
		id string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty map",
			args{
				map[string]int64{},
				"channel",
			},
			"",
		},
		{
			"not found",
			args{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel2",
			},
			"",
		},
		{

			"found",
			args{
				map[string]int64{
					"channel": 1638494510037400,
				},
				"channel",
			},
			"1638494510.037400",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := latest(tt.args.m, tt.args.id); got != tt.want {
				t.Errorf("latest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_Save(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"valid",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				filename: "test.json",
			},
			false,
		},
		{
			"with values",
			fields{
				Version: 0.1,
				Channels: map[_id]int64{
					"channel": 1638494510037400,
				},
				Threads: map[_idAndThread]int64{
					"channel:thread": 1638494510037400,
				},
				Files: map[_id]_id{
					"file": "channel",
				},
			},
			args{
				filename: "test.json",
			},
			false,
		},
		{
			"invalid filename",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				filename: "../..../..././...../",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			dir := t.TempDir()
			fullpath := filepath.Join(dir, tt.args.filename)
			if err := s.Save(fullpath); (err != nil) != tt.wantErr {
				t.Errorf("State.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			fi, err := os.Stat(fullpath)
			if err != nil {
				t.Errorf("State.Save() error = %v", err)
			}
			if fi == nil {
				return
			}
			if fi.Size() == 0 {
				t.Errorf("State.Save() error = %v", err)
			}
			s2, err := Load(fullpath)
			if err != nil {
				t.Errorf("State.Load() error = %v", err)
			}
			if !assert.Equal(t, s, s2) {
				t.Error("State.Load() values mismatch")
			}
		})
	}
}

func TestState_AddMessage(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		channelID string
		messageTS string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantMap map[_id]int64
	}{
		{
			"empty (shoudn't panic)",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				channelID: "channel",
				messageTS: "1638494510.037400",
			},
			map[_id]int64{
				"channel": 1638494510037400,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			s.AddMessage(tt.args.channelID, tt.args.messageTS)
		})
	}
}

func Test_load(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *State
		wantErr bool
	}{
		{
			"empty",
			args{
				r: strings.NewReader("{}"),
			},
			&State{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			false,
		},
		{
			"valid",
			args{
				r: strings.NewReader(`{
					"version": 0.1,
					"channels": {
						"channel": 1638494510037400
					},
					"threads": {
						"channel:thread": 1638494510037400
					},
					"files": {
						"file": "channel"
					}
				}`),
			},
			&State{
				Version: 0.1,
				Channels: map[_id]int64{
					"channel": 1638494510037400,
				},
				Threads: map[_idAndThread]int64{
					"channel:thread": 1638494510037400,
				},
				Files: map[_id]_id{
					"file": "channel",
				},
			},
			false,
		},
		{
			"invalid version",
			args{
				r: strings.NewReader(`{
					"version": 1.1,
					"channels": {
						"channel": 1638494510037400
					},
					"threads": {
						"channel:thread": 1638494510037400
					},
					"files": {
						"file": "channel"
					}
				}`),
			},
			nil,
			true,
		},
		{
			"invalid json",
			args{
				r: strings.NewReader(`this is not a json, but some bullshit`),
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := load(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("load() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want *State
	}{
		{
			"new",
			args{"x"},
			&State{
				Version:       Version,
				ChunkFilename: "x",
				Channels:      make(map[_id]int64),
				Threads:       make(map[_idAndThread]int64),
				Files:         make(map[_id]_id),
			},
		},
		{
			"empty filename",
			args{""},
			&State{
				Version:       Version,
				ChunkFilename: "",
				Channels:      make(map[_id]int64),
				Threads:       make(map[_idAndThread]int64),
				Files:         make(map[_id]_id),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.filename); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_AddThread(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		channelID string
		threadTS  string
		ts        string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[_idAndThread]int64
	}{
		{
			"empty (shoudn't panic)",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "1638494510.037400",
				ts:        "1638494510.037401",
			},
			map[_idAndThread]int64{
				"channel:1638494510.037400": 1638494510037401,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			s.AddThread(tt.args.channelID, tt.args.threadTS, tt.args.ts)
			if !reflect.DeepEqual(s.Threads, tt.want) {
				t.Errorf("State.AddThread() = %v, want %v", s.Threads, tt.want)
			}
		})
	}
}

func TestState_AddFile(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		channelID string
		fileID    string
		path      string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[_id]_id
	}{
		{
			"empty (shoudn't panic)",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				channelID: "channel",
				fileID:    "file",
				path:      "path",
			},
			map[_id]_id{
				"channel:file": "path",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			s.AddFile(tt.args.channelID, tt.args.fileID, tt.args.path)
			if !reflect.DeepEqual(s.Files, tt.want) {
				t.Errorf("State.AddFile() = %v, want %v", s.Files, tt.want)
			}
		})
	}
}

func TestState_HasChannel(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"empty",
			fields{
				Version:  Version,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel",
			},
			false,
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: map[_id]int64{"channel": 1},
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel2",
			},
			false,
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: map[_id]int64{"channel": 1},
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.HasChannel(tt.args.id); got != tt.want {
				t.Errorf("State.HasChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_HasThread(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		channelID string
		threadTS  string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"empty",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "1638494510.037400",
			},
			false,
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  map[_idAndThread]int64{"channel:1638494510.037400": 1638494510037401},
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "1638494510.037401",
			},
			false,
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  map[_idAndThread]int64{"channel:1638494510.037400": 1638494510037401},
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "1638494510.037400",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.HasThread(tt.args.channelID, tt.args.threadTS); got != tt.want {
				t.Errorf("State.HasThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_HasFile(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"empty",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "file",
			},
			false,
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    map[_id]_id{"file": "file"},
			},
			args{
				id: "file2",
			},
			false,
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    map[_id]_id{"file": "file"},
			},
			args{
				id: "file",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.HasFile(tt.args.id); got != tt.want {
				t.Errorf("State.HasFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_threadID(t *testing.T) {
	type args struct {
		channelID string
		threadTS  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{
				channelID: "",
				threadTS:  "",
			},
			":",
		},
		{
			"not empty",
			args{
				channelID: "channel",
				threadTS:  "123",
			},
			"channel:123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := threadID(tt.args.channelID, tt.args.threadTS); got != tt.want {
				t.Errorf("threadID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_LatestChannelTS(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			"empty",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel",
			},
			"",
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: map[_id]int64{"channel": 1638494510037401},
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel2",
			},
			"",
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: map[_id]int64{"channel": 1638494510037401},
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "channel",
			},
			"1638494510.037401",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.LatestChannelTS(tt.args.id); got != tt.want {
				t.Errorf("State.LatestChannelTS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_LatestThreadTS(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		channelID string
		threadTS  string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			"empty",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				channelID: "",
				threadTS:  "",
			},
			"",
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  map[_idAndThread]int64{"channel:123": 1638494510037401},
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "321",
			},
			"",
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  map[_idAndThread]int64{"channel:123": 1638494510037401},
				Files:    nil,
			},
			args{
				channelID: "channel",
				threadTS:  "123",
			},
			"1638494510.037401",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.LatestThreadTS(tt.args.channelID, tt.args.threadTS); got != tt.want {
				t.Errorf("State.LatestThreadTS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_FileChannelID(t *testing.T) {
	type fields struct {
		Version  float64
		Channels map[_id]int64
		Threads  map[_idAndThread]int64
		Files    map[_id]_id
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			"empty",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    nil,
			},
			args{
				id: "file",
			},
			"",
		},
		{
			"not empty, not exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    map[_id]_id{"file": "channel:123"},
			},
			args{
				id: "file2",
			},
			"",
		},
		{
			"not empty, exists",
			fields{
				Version:  0.1,
				Channels: nil,
				Threads:  nil,
				Files:    map[_id]_id{"file": "channel:123"},
			},
			args{
				id: "file",
			},
			"channel:123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &State{
				Version:  tt.fields.Version,
				Channels: tt.fields.Channels,
				Threads:  tt.fields.Threads,
				Files:    tt.fields.Files,
			}
			if got := s.FileChannelID(tt.args.id); got != tt.want {
				t.Errorf("State.FileChannelID() = %v, want %v", got, tt.want)
			}
		})
	}
}
