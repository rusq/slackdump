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
	"github.com/rusq/slackdump/v3/source"
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
				ctx:    t.Context(),
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
			src := source.OpenChunkDir(cd, true)
			cvt := ExpConverter{
				src: src,
				fsa: tt.args.fsa,
			}
			if err := cvt.Convert(tt.args.ctx, tt.args.id, ""); (err != nil) != tt.wantErr {
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

// func TestExpConverter_writeMessages(t *testing.T) {
// 	type fields struct {
// 		// src     source.Sourcer
// 		// fsa     fsadapter.FS
// 		users   atomic.Value
// 		msgFunc []msgUpdFunc
// 	}
// 	type args struct {
// 		ctx context.Context
// 		ci  *slack.Channel
// 	}
// 	tests := []struct {
// 		name      string
// 		fields    fields
// 		expectFn  func(ms *mock_source.MockSourcer, mst *mock_source.MockStorage)
// 		args      args
// 		wantFiles map[string]testutil.FileInfo
// 		wantErr   bool
// 	}{
// 		{
// 			name:   "threaded messages",
// 			fields: fields{},
// 			args: args{
// 				ctx: context.Background(),
// 				ci:  fixtures.Load[[]*slack.Channel](fixtures.TestChannels)[0],
// 			},
// 			expectFn: func(ms *mock_source.MockSourcer, mst *mock_source.MockStorage) {
// 				chanmsg := testutil.Slice2Seq2(
// 					fixtures.Load[[]slack.Message](fixtures.ConvertPublic1AllMessagesJSON),
// 				)
// 				threadmsg := testutil.Slice2Seq2(
// 					fixtures.Load[[]slack.Message](fixtures.ConvertPublic1AllThreadMessagesJSON),
// 				)
// 				ms.EXPECT().Sorted(gomock.Any(), gomock.Any, false, gomock.Any()).
// 					Return(nil)
// 				ms.EXPECT().AllThreadMessages(gomock.Any(), gomock.Any(), gomock.Any()).
// 					Return(threadmsg, nil)
// 			},
// 			wantFiles: map[string]testutil.FileInfo{
// 				"random/2025-01-10.json": {Name: "2025-01-10.json", Size: 1892},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctrl := gomock.NewController(t)
// 			ms := mock_source.NewMockSourcer(ctrl)
// 			mst := mock_source.NewMockStorage(ctrl)
// 			if tt.expectFn != nil {
// 				tt.expectFn(ms, mst)
// 			}
//
// 			dir := t.TempDir()
// 			fsa := fsadapter.NewDirectory(dir)
//
// 			e := &ExpConverter{
// 				src:     ms,
// 				fsa:     fsa,
// 				users:   tt.fields.users,
// 				msgFunc: tt.fields.msgFunc,
// 			}
// 			if err := e.writeMessages(tt.args.ctx, tt.args.ci); (err != nil) != tt.wantErr {
// 				t.Errorf("ExpConverter.writeMessages() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			gotfiles := testutil.CollectFiles(t, os.DirFS(dir))
// 			assert.Equal(t, tt.wantFiles, gotfiles)
// 		})
// 	}
// }

func TestExpConverter_newAccumulator(t *testing.T) {
	type fields struct {
		src     source.Sourcer
		fsa     fsadapter.FS
		users   atomic.Value
		msgFunc []msgUpdFunc
	}
	type args struct {
		ctx     context.Context
		channel *slack.Channel
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *expmsgAccum
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExpConverter{
				src:     tt.fields.src,
				fsa:     tt.fields.fsa,
				users:   tt.fields.users,
				msgFunc: tt.fields.msgFunc,
			}
			if got := e.newAccumulator(tt.args.ctx, tt.args.channel); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpConverter.newAccumulator() = %v, want %v", got, tt.want)
			}
		})
	}
}
