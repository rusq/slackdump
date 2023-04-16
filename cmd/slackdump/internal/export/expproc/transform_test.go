package expproc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/fixtures/fixchunks"
	"github.com/slack-go/slack"
)

func Test_transform(t *testing.T) {
	// TODO: automate.
	// MANUAL
	const (
		base   = "../../../../../"
		srcdir = base + "tmp/exportv3"
		fsaDir = base + "tmp/exportv3/out"
	)
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
			if err := transform(tt.args.ctx, tt.args.fsa, tt.args.srcdir, tt.args.id, nil); (err != nil) != tt.wantErr {
				t.Errorf("transform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_readChanInfo(t *testing.T) {
	type args struct {
		r io.ReadSeeker
	}
	tests := []struct {
		name    string
		args    args
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				r: marshalChunks(
					fixchunks.TestPublicChannelInfo,
					fixchunks.TestPublicChannelMessages,
				),
			},
			want: []slack.Channel{
				*fixchunks.TestPublicChannelInfo.Channel,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readChanInfo(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readChanInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readChanInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func marshalChunks(chunks ...chunk.Chunk) io.ReadSeeker {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, c := range chunks {
		if err := enc.Encode(c); err != nil {
			panic(err)
		}
	}
	return bytes.NewReader(b.Bytes())
}
