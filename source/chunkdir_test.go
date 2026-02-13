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
	"context"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/directory"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/fixtures"
)

func testDir(t *testing.T) *chunk.Directory {
	t.Helper()
	dir := t.TempDir()
	d, err := chunk.OpenDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

type nopFiler struct{}

func (n nopFiler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	return nil
}
func (n nopFiler) Close() error { return nil }

type chunkPrepFn func(t *testing.T, ctx context.Context, d *chunk.Directory)

var testChannelInfo = fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON)[0]

func TestChunkDir_ChannelInfo(t *testing.T) {
	type fields struct {
		d       *chunk.Directory
		fast    bool
		files   Storage
		avatars Storage
	}
	type args struct {
		in0       context.Context
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  chunkPrepFn
		want    *slack.Channel
		wantErr bool
	}{
		{
			name: "channel info is in the channel file",
			fields: fields{
				d: testDir(t),
			},
			args: args{
				channelID: testChannelInfo.ID,
			},
			prepFn: func(t *testing.T, ctx context.Context, d *chunk.Directory) {
				p, err := directory.NewConversation(d, nopFiler{}, &chunk.NopTransformer{})
				if err != nil {
					t.Fatal(err)
				}
				defer p.Close()
				if err := p.ChannelInfo(ctx, &testChannelInfo, ""); err != nil {
					t.Fatal(err)
				}
			},
			want:    &testChannelInfo,
			wantErr: false,
		},
		{
			name: "channel info is in the thread file",
			fields: fields{
				d: testDir(t),
			},
			args: args{
				channelID: testChannelInfo.ID,
			},
			prepFn: func(t *testing.T, ctx context.Context, d *chunk.Directory) {
				p, err := directory.NewConversation(d, nopFiler{}, &chunk.NopTransformer{})
				if err != nil {
					t.Fatal(err)
				}
				defer p.Close()
				if err := p.ChannelInfo(ctx, &testChannelInfo, "123456.789"); err != nil {
					t.Fatal(err)
				}
			},
			want:    &testChannelInfo,
			wantErr: false,
		},
		{
			name: "no relevant data in the file",
			fields: fields{
				d: testDir(t),
			},
			args: args{
				channelID: testChannelInfo.ID,
			},
			prepFn: func(t *testing.T, ctx context.Context, d *chunk.Directory) {
				p, err := directory.NewConversation(d, nopFiler{}, &chunk.NopTransformer{})
				if err != nil {
					t.Fatal(err)
				}
				defer p.Close()
				mm := fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)
				if err := p.Messages(ctx, testChannelInfo.ID, 0, true, mm); err != nil {
					t.Fatal(err)
				}
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "file open error (empty dir)",
			fields: fields{
				d: testDir(t),
			},
			args: args{
				channelID: testChannelInfo.ID,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, t.Context(), tt.fields.d)
			}
			c := &ChunkDir{
				d:       tt.fields.d,
				fast:    tt.fields.fast,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
			}
			got, err := c.ChannelInfo(tt.args.in0, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChunkDir.ChannelInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChunkDir.ChannelInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkDir_channelInfo(t *testing.T) {
	type fields struct {
		d       *chunk.Directory
		fast    bool
		files   Storage
		avatars Storage
	}
	type args struct {
		fileID chunk.FileID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *slack.Channel
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ChunkDir{
				d:       tt.fields.d,
				fast:    tt.fields.fast,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
			}
			got, err := c.channelInfo(tt.args.fileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChunkDir.channelInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChunkDir.channelInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
