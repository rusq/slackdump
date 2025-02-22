package transform

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/source/mock_source"
	"github.com/rusq/slackdump/v3/internal/testutil"
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

func TestExpConverter_writeMessages(t *testing.T) {
	type fields struct {
		// src     source.Sourcer
		// fsa     fsadapter.FS
		users   atomic.Value
		msgFunc []msgUpdFunc
	}
	type args struct {
		ctx context.Context
		ci  *slack.Channel
	}
	tests := []struct {
		name      string
		fields    fields
		expectFn  func(ms *mock_source.MockSourcer, mst *mock_source.MockStorage)
		args      args
		wantFiles map[string]testutil.FileInfo
		wantErr   bool
	}{
		{
			name:   "threaded messages",
			fields: fields{},
			args: args{
				ctx: context.Background(),
				ci:  fixtures.Load[[]*slack.Channel](fixtures.TestChannels)[0],
			},
			expectFn: func(ms *mock_source.MockSourcer, mst *mock_source.MockStorage) {
				chanmsg := testutil.Slice2Seq2(
					fixtures.Load[[]slack.Message](fixtures.ConvertPublic1AllMessagesJSON),
				)
				threadmsg := testutil.Slice2Seq2(
					fixtures.Load[[]slack.Message](fixtures.ConvertPublic1AllThreadMessagesJSON),
				)
				ms.EXPECT().AllMessages(gomock.Any(), gomock.Any()).
					Return(chanmsg, nil)
				ms.EXPECT().AllThreadMessages(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(threadmsg, nil)
			},
			wantFiles: map[string]testutil.FileInfo{
				"random/2025-01-10.json": {Name: "2025-01-10.json", Size: 1892},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_source.NewMockSourcer(ctrl)
			mst := mock_source.NewMockStorage(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mst)
			}

			dir := t.TempDir()
			fsa := fsadapter.NewDirectory(dir)

			e := &ExpConverter{
				src:     ms,
				fsa:     fsa,
				users:   tt.fields.users,
				msgFunc: tt.fields.msgFunc,
			}
			if err := e.writeMessages(tt.args.ctx, tt.args.ci); (err != nil) != tt.wantErr {
				t.Errorf("ExpConverter.writeMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
			gotfiles := testutil.CollectFiles(t, os.DirFS(dir))
			assert.Equal(t, tt.wantFiles, gotfiles)
		})
	}
}
