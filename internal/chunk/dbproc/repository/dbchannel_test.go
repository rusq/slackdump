package repository

import (
	"context"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

var (
	// channel names are deliberately out of order to check sorting.
	ch100 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel C 100", Conversation: slack.Conversation{ID: "C100"}}}
	ch200 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel B 200", Conversation: slack.Conversation{ID: "C200"}}}
	ch300 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel D 300", Conversation: slack.Conversation{ID: "C300"}}}
	ch400 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel A 400", Conversation: slack.Conversation{ID: "C400"}}}

	chi100 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel C 100", Conversation: slack.Conversation{ID: "C100"}}}
	chi200 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel B 200", Conversation: slack.Conversation{ID: "C200"}}}     // channel info
	chi300 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel D 300 new", Conversation: slack.Conversation{ID: "C300"}}} // channel info
	chi400 = &slack.Channel{GroupConversation: slack.GroupConversation{Name: "channel A 400", Conversation: slack.Conversation{ID: "C400"}}}     // channel info

	dbch100, _ = NewDBChannel(1, 0, ch100) // these all belong to the same chunk.
	dbch200, _ = NewDBChannel(1, 1, ch200) //
	dbch300, _ = NewDBChannel(1, 2, ch300) //
	dbch400, _ = NewDBChannel(1, 3, ch400) //

	dbchi100, _ = NewDBChannel(2, 0, chi100) // there's only one channel info per chunk.
	dbchi200, _ = NewDBChannel(3, 0, chi200) //
	dbchi300, _ = NewDBChannel(4, 0, chi300) //
	dbchi400, _ = NewDBChannel(5, 0, chi400) //
)

func prepChannels(t *testing.T, conn PrepareExtContext) {
	ctx := context.Background()
	prepChunk(chunk.CChannels, chunk.CChannelInfo, chunk.CChannelInfo, chunk.CChannelInfo, chunk.CChannelInfo)(t, conn)
	cr := NewChannelRepository()
	err := cr.Insert(ctx, conn, dbch100, dbch200, dbch300, dbch400)
	require.NoError(t, err)
	err = cr.Insert(ctx, conn, dbchi100, dbchi200, dbchi300, dbchi400) // insert channel info
	require.NoError(t, err)
}

func TestNewDBChannel(t *testing.T) {
	type args struct {
		chunkID int64
		n       int
		channel *slack.Channel
	}
	tests := []struct {
		name    string
		args    args
		want    *DBChannel
		wantErr bool
	}{
		{
			name: "creates a new DBChannel",
			args: args{
				chunkID: 1,
				n:       50,
				channel: ch100,
			},
			want: &DBChannel{
				ID:      "C100",
				ChunkID: 1,
				Name:    ptr("channel C 100"),
				Index:   50,
				Data:    must(marshal(ch100)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBChannel(tt.args.chunkID, tt.args.n, tt.args.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_channelRepository_AllOfType(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBChannel]
	}
	type args struct {
		ctx    context.Context
		conn   sqlx.QueryerContext
		typeID []chunk.ChunkType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    []testResult[DBChannel]
		wantErr bool
	}{
		{
			name: "returns most recent versions in the correct order",
			fields: fields{
				genericRepository: genericRepository[DBChannel]{t: DBChannel{}},
			},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: []chunk.ChunkType{chunk.CChannelInfo},
			},
			prepFn: prepChannels,
			want: []testResult[DBChannel]{
				{V: *dbchi400, Err: nil},
				{V: *dbchi200, Err: nil},
				{V: *dbchi100, Err: nil},
				{V: *dbchi300, Err: nil},
			},
		},
		{
			name: "selecting channels",
			fields: fields{
				genericRepository: genericRepository[DBChannel]{t: DBChannel{}},
			},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: []chunk.ChunkType{chunk.CChannels},
			},
			prepFn: prepChannels,
			want: []testResult[DBChannel]{
				{V: *dbch400, Err: nil},
				{V: *dbch200, Err: nil},
				{V: *dbch100, Err: nil},
				{V: *dbch300, Err: nil},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := channelRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.AllOfType(tt.args.ctx, tt.args.conn, tt.args.typeID...)
			if (err != nil) != tt.wantErr {
				t.Errorf("channelRepository.AllOfType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assertIterResult(t, tt.want, got)
		})
	}
}
