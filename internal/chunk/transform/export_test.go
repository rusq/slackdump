package transform

import (
	"context"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/source"
)

func Test_transform(t *testing.T) {
	// TODO: automate.
	// MANUAL
	var (
		base   = filepath.Join("..", "..", "..")
		srcdir = filepath.Join(base, "tmp", "exportv3")
	)
	fixtures.SkipIfNotExist(t, srcdir)
	fsaDir := t.TempDir()
	type args struct {
		ctx    context.Context
		fsa    fsadapter.FS
		srcdir string
		id     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				ctx:    context.Background(),
				fsa:    fsadapter.NewDirectory(fsaDir),
				srcdir: srcdir,
				// id:     "D01MN4X7UGP",
				id: "C01SPFM1KNY",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd, err := chunk.OpenDir(tt.args.srcdir)
			if err != nil {
				t.Fatal(err)
			}
			defer cd.Close()
			src := source.NewChunkDir(cd, true)
			cvt := ExpConverter{
				src: src,
				fsa: tt.args.fsa,
			}
			if err := cvt.Convert(tt.args.ctx, chunk.FileID(tt.args.id)); (err != nil) != tt.wantErr {
				t.Errorf("transform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpConverter_getUsers(t *testing.T) {
	setUsers := func(uu []slack.User) *atomic.Value {
		var v atomic.Value
		v.Store(uu)
		return &v
	}
	type fields struct {
		fsa     fsadapter.FS
		users   atomic.Value
		msgFunc []msgUpdFunc
	}
	tests := []struct {
		name   string
		fields fields
		want   []slack.User
	}{
		{
			name: "existing users",
			fields: fields{
				users: *setUsers(fixtures.TestUsers),
			},
			want: fixtures.TestUsers,
		},
		{
			name: "no users",
			fields: fields{
				users: *setUsers(nil),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExpConverter{
				fsa:     tt.fields.fsa,
				users:   tt.fields.users,
				msgFunc: tt.fields.msgFunc,
			}
			if got := e.getUsers(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpConverter.getUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}
