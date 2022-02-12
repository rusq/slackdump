package slackdump

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func Test_sortMessages(t *testing.T) {
	type args struct {
		msgs []Message
	}
	tests := []struct {
		name     string
		args     args
		wantMsgs []Message
	}{
		{
			"empty",
			args{[]Message{}},
			[]Message{},
		},
		{
			"sort ok",
			args{[]Message{
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425514",
				}}},
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425511",
				}}},
			}},
			[]Message{
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425511",
				}}},
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425514",
				}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortMessages(tt.args.msgs)
			assert.Equal(t, tt.wantMsgs, tt.args.msgs)
		})
	}
}

var (
	testMsg1  = slack.Message{Msg: slack.Msg{ClientMsgID: "a", Type: "x"}}
	testMsg2  = slack.Message{Msg: slack.Msg{ClientMsgID: "b", Type: "y"}}
	testMsg3  = slack.Message{Msg: slack.Msg{ClientMsgID: "c", Type: "z"}}
	testMsg4t = slack.Message{Msg: slack.Msg{ClientMsgID: "c", Type: "z", ThreadTimestamp: "d"}}
)

func TestSlackDumper_convertMsgs(t *testing.T) {

	type args struct {
		sm []slack.Message
	}
	tests := []struct {
		name string
		args args
		want []Message
	}{
		{
			"ok",
			args{[]slack.Message{
				testMsg1,
				testMsg2,
				testMsg3,
			}},
			[]Message{
				{Message: testMsg1},
				{Message: testMsg2},
				{Message: testMsg3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{}
			if got := sd.convertMsgs(tt.args.sm); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.convertMsgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_DumpMessages(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
		options   options
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(c *mockClienter)
		want     *Conversation
		wantErr  bool
	}{
		{
			"all ok",
			fields{},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				c.EXPECT().GetConversationHistoryContext(
					gomock.Any(),
					&slack.GetConversationHistoryParameters{
						ChannelID: "CHANNEL",
					}).Return(
					&slack.GetConversationHistoryResponse{
						SlackResponse: slack.SlackResponse{Ok: true},
						Messages: []slack.Message{
							testMsg1,
							testMsg2,
							testMsg3,
						},
					},
					nil)
			},
			&Conversation{
				ID: "CHANNEL",
				Messages: []Message{
					{Message: testMsg1},
					{Message: testMsg2},
					{Message: testMsg3},
				}},
			false,
		},
		{
			"iteration test",
			fields{},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				first := c.EXPECT().
					GetConversationHistoryContext(
						gomock.Any(),
						&slack.GetConversationHistoryParameters{
							ChannelID: "CHANNEL",
						}).
					Return(
						&slack.GetConversationHistoryResponse{
							HasMore:       true,
							SlackResponse: slack.SlackResponse{Ok: true},
							ResponseMetaData: struct {
								NextCursor string "json:\"next_cursor\""
							}{"cur"},
							Messages: []slack.Message{
								testMsg1,
							},
						},
						nil,
					)

				c.EXPECT().
					GetConversationHistoryContext(
						gomock.Any(),
						&slack.GetConversationHistoryParameters{
							ChannelID: "CHANNEL",
							Cursor:    "cur",
						}).
					Return(
						&slack.GetConversationHistoryResponse{
							SlackResponse: slack.SlackResponse{Ok: true},
							Messages: []slack.Message{
								testMsg2,
							},
						},
						nil,
					).
					After(first)
			},
			&Conversation{
				ID: "CHANNEL",
				Messages: []Message{
					{Message: testMsg1},
					{Message: testMsg2},
				}},
			false,
		},
		{
			"resp not ok",
			fields{},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				c.EXPECT().GetConversationHistoryContext(
					gomock.Any(),
					gomock.Any(),
				).Return(
					&slack.GetConversationHistoryResponse{
						SlackResponse: slack.SlackResponse{Ok: false},
					},
					nil)
			},
			nil,
			true,
		},
		{
			"sudden bleep bloop error",
			fields{},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				c.EXPECT().GetConversationHistoryContext(
					gomock.Any(),
					gomock.Any(),
				).Return(
					nil,
					errors.New("bleep bloop gtfo"))
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
			got, err := sd.DumpMessages(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.DumpMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlackDumper_dumpThread(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
		options   options
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
			args{context.Background(), newLimiter(noTier, 1, 0), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
					).
					Return(
						[]slack.Message{testMsg1, testMsg2, testMsg3},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]Message{{Message: testMsg1}, {Message: testMsg2}, {Message: testMsg3}},
			false,
		},
		{
			"iterating over",
			fields{},
			args{context.Background(), newLimiter(noTier, 1, 0), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
					).
					Return(
						[]slack.Message{testMsg1},
						true,
						"blah",
						nil,
					).
					Times(1)
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah"},
					).
					Return(
						[]slack.Message{testMsg2},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]Message{{Message: testMsg1}, {Message: testMsg2}},
			false,
		},
		{
			"sudden bleep bloop error",
			fields{},
			args{context.Background(), newLimiter(noTier, 1, 0), "CHANNEL", "THREADTS"},
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
				l:         newLimiter(noTier, 1, 0),
				msgs:      []Message{{Message: testMsg1}},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string) ([]Message, error) {
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
				l:         newLimiter(noTier, 1, 0),
				msgs:      []Message{{Message: testMsg1}, {Message: testMsg4t}},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string) ([]Message, error) {
					return []Message{{Message: testMsg4t}, {Message: testMsg2}}, nil
				},
			},
			1,
			false,
		},
		{
			"skipping empty messages",
			args{
				ctx:       context.Background(),
				l:         newLimiter(noTier, 1, 0),
				msgs:      []Message{{Message: testMsg4t}, {Message: testMsg1}},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string) ([]Message, error) {
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
				l:         newLimiter(noTier, 1, 0),
				msgs:      []Message{{Message: testMsg4t}},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string) ([]Message, error) {
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
