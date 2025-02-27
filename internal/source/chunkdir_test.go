package source

import (
	"context"
	"reflect"
	"testing"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/fixtures"
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

type nopTransformer struct{}

func (n nopTransformer) Transform(ctx context.Context, id chunk.FileID) error { return nil }

type chunkPrepFn func(t *testing.T, ctx context.Context, d *chunk.Directory)

var testChannelInfo = fixtures.Load[[]slack.Channel](fixtures.TestChannels)[0]

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
				p, err := dirproc.NewConversation(d, nopFiler{}, nopTransformer{})
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
				p, err := dirproc.NewConversation(d, nopFiler{}, nopTransformer{})
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
				p, err := dirproc.NewConversation(d, nopFiler{}, nopTransformer{})
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
				tt.prepFn(t, context.Background(), tt.fields.d)
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
