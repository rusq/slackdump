package slackdump

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/network"
)

func TestSlackDumper_DumpThread(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		ctx       context.Context
		channelID string
		threadTS  string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mockClienter)
		want     *Conversation
		wantErr  bool
	}{
		{"chan and thread are empty", fields{options: DefOptions}, args{context.Background(), "", ""}, nil, nil, true},
		{"thread empty", fields{options: DefOptions}, args{context.Background(), "xxx", ""}, nil, nil, true},
		{"chan empty", fields{options: DefOptions}, args{context.Background(), "", "yyy"}, nil, nil, true},
		{
			"ok",
			fields{options: DefOptions},
			args{context.Background(), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Limit: DefOptions.RepliesPerReq},
					).
					Return(
						[]slack.Message{testMsg1.Message, testMsg2.Message, testMsg3.Message},
						false,
						"",
						nil,
					).
					Times(1)
				mockConvInfo(mc, "CHANNEL", "channel_name")
			},
			&Conversation{Name: "channel_name", ID: "CHANNEL", ThreadTS: "THREAD", Messages: []Message{testMsg1, testMsg2, testMsg3}},
			false,
		},
		{
			"iterating over",
			fields{options: DefOptions},
			args{context.Background(), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Limit: DefOptions.RepliesPerReq},
					).
					Return(
						[]slack.Message{testMsg1.Message},
						true,
						"blah",
						nil,
					).
					Times(1)
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah", Limit: DefOptions.RepliesPerReq},
					).
					Return(
						[]slack.Message{testMsg2.Message},
						false,
						"",
						nil,
					).
					Times(1)
				mockConvInfo(mc, "CHANNEL", "channel_name")
			},
			&Conversation{Name: "channel_name", ID: "CHANNEL", ThreadTS: "THREAD", Messages: []Message{testMsg1, testMsg2}},
			false,
		},
		{
			"sudden bleep bloop error",
			fields{options: DefOptions},
			args{context.Background(), "CHANNEL", "THREADTS"},
			func(c *mockClienter) {
				c.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						gomock.Any(),
					).
					Return(
						nil,
						false,
						"",
						errors.New("bleep bloop gtfo"),
					).
					Times(1)
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := newmockClienter(ctrl)

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.DumpThread(tt.args.ctx, tt.args.channelID, tt.args.threadTS)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.dumpThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlackDumper_populateThreads(t *testing.T) {
	type args struct {
		ctx       context.Context
		l         *rate.Limiter
		msgs      []Message
		channelID string
		dumpFn    threadFunc
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			"ok, no threads",
			args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []Message{testMsg1},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, processFn ...ProcessFunc) ([]Message, error) {
					return nil, nil
				},
			},
			0,
			false,
		},
		{
			"ok, thread",
			args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []Message{testMsg1, testMsg4t},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, processFn ...ProcessFunc) ([]Message, error) {
					return []Message{testMsg4t, testMsg2}, nil
				},
			},
			1,
			false,
		},
		{
			"skipping empty messages",
			args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []Message{testMsg4t, testMsg1},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, processFn ...ProcessFunc) ([]Message, error) {
					return []Message{}, nil
				},
			},
			0,
			false,
		},
		{
			"failing on dumpFn returning error",
			args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []Message{testMsg4t},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, processFn ...ProcessFunc) ([]Message, error) {
					return nil, errors.New("bam")
				},
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{}
			got, err := sd.populateThreads(tt.args.ctx, tt.args.l, tt.args.msgs, tt.args.channelID, tt.args.dumpFn)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.populateThreads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackDumper.populateThreads() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_dumpThread(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		ctx       context.Context
		l         *rate.Limiter
		channelID string
		threadTS  string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mockClienter)
		want     []Message
		wantErr  bool
	}{
		{
			"ok",
			fields{},
			args{context.Background(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
					).
					Return(
						[]slack.Message{testMsg1.Message, testMsg2.Message, testMsg3.Message},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]Message{testMsg1, testMsg2, testMsg3},
			false,
		},
		{
			"iterating over",
			fields{options: DefOptions},
			args{context.Background(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Limit: DefOptions.RepliesPerReq},
					).
					Return(
						[]slack.Message{testMsg1.Message},
						true,
						"blah",
						nil,
					).
					Times(1)
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah", Limit: DefOptions.RepliesPerReq},
					).
					Return(
						[]slack.Message{testMsg2.Message},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]Message{testMsg1, testMsg2},
			false,
		},
		{
			"sudden bleep bloop error",
			fields{options: DefOptions},
			args{context.Background(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREADTS"},
			func(c *mockClienter) {
				c.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						gomock.Any(),
					).
					Return(
						nil,
						false,
						"",
						errors.New("bleep bloop gtfo"),
					).
					Times(1)
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := newmockClienter(ctrl)

			tt.expectFn(mc)

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.dumpThread(tt.args.ctx, tt.args.l, tt.args.channelID, tt.args.threadTS)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.dumpThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
