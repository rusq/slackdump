package slackdump

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

var (
	testMsg1 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "d1831c57-3b7f-4a0c-ab9a-a18d4a58a01c",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497751.040300",
		Text:        "Test message \u0026lt; \u0026gt; \u0026lt; \u0026gt;",
	}}}
	testMsg2 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "b11431d3-a5c4-4612-b09c-b074e9ddace7",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497781.040300",
		Text:        "message 2",
	}}}
	testMsg3 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "a99df2f2-1fd6-421f-9453-6903974b683a",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1641541791.000000",
		Text:        "message 3",
	}}}
	testMsg4t = types.Message{
		Message: slack.Message{Msg: slack.Msg{
			ClientMsgID:     "931db474-6ea8-43bc-9ff7-804309716ded",
			Type:            "message",
			User:            "UP58RAHCJ",
			Timestamp:       "1638524854.042000",
			ThreadTimestamp: "1638524854.042000",
			ReplyCount:      3,
			Text:            "message 4",
		}},
		ThreadReplies: []types.Message{
			{Message: slack.Message{Msg: slack.Msg{
				ClientMsgID:     "a99df2f2-1fd6-421f-9453-6903974b683a",
				Type:            "message",
				Timestamp:       "1638554726.042700",
				ThreadTimestamp: "1638524854.042000",
				User:            "U01HPAR0YFN",
				Text:            "blah blah, reply 1",
			}}},
		},
	}
)

func TestSession_DumpMessages(t *testing.T) {
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mockClienter)
		want     *types.Conversation
		wantErr  bool
	}{
		{
			"all ok",
			fields{options: DefOptions},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				c.EXPECT().GetConversationHistoryContext(
					gomock.Any(),
					&slack.GetConversationHistoryParameters{
						ChannelID: "CHANNEL",
						Limit:     DefOptions.ConversationsPerReq,
						Inclusive: true,
					}).Return(
					&slack.GetConversationHistoryResponse{
						SlackResponse: slack.SlackResponse{Ok: true},
						Messages: []slack.Message{
							testMsg1.Message,
							testMsg2.Message,
							testMsg3.Message,
						},
					},
					nil)
				mockConvInfo(c, "CHANNEL", "channel_name")
			},
			&types.Conversation{
				Name: "channel_name",
				ID:   "CHANNEL",
				Messages: []types.Message{
					testMsg1,
					testMsg2,
					testMsg3,
				}},
			false,
		},
		{
			"channelID is empty",
			fields{options: DefOptions},
			args{context.Background(), ""},
			func(c *mockClienter) {},
			nil,
			true,
		},
		{
			"iteration test",
			fields{options: DefOptions},
			args{context.Background(), "CHANNEL"},
			func(c *mockClienter) {
				first := c.EXPECT().
					GetConversationHistoryContext(
						gomock.Any(),
						&slack.GetConversationHistoryParameters{
							ChannelID: "CHANNEL",
							Limit:     DefOptions.ConversationsPerReq,
							Inclusive: true,
						}).
					Return(
						&slack.GetConversationHistoryResponse{
							HasMore:       true,
							SlackResponse: slack.SlackResponse{Ok: true},
							ResponseMetaData: struct {
								NextCursor string "json:\"next_cursor\""
							}{"cur"},
							Messages: []slack.Message{
								testMsg1.Message,
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
							Limit:     DefOptions.ConversationsPerReq,
							Inclusive: true,
						}).
					Return(
						&slack.GetConversationHistoryResponse{
							SlackResponse: slack.SlackResponse{Ok: true},
							Messages: []slack.Message{
								testMsg2.Message,
							},
						},
						nil,
					).
					After(first)
				mockConvInfo(c, "CHANNEL", "channel_name")
			},
			&types.Conversation{
				Name: "channel_name",
				ID:   "CHANNEL",
				Messages: []types.Message{
					testMsg1,
					testMsg2,
				}},
			false,
		},
		{
			"resp not ok",
			fields{options: DefOptions},
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
			fields{options: DefOptions},
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

			sd := &Session{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.DumpAll(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.DumpMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSession_DumpAll(t *testing.T) {
	t.Parallel()
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		ctx      context.Context
		slackURL string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(sc *mockClienter)
		want     *types.Conversation
		wantErr  bool
	}{
		{
			name:   "conversation url",
			fields: fields{options: DefOptions},
			args:   args{context.Background(), "https://ora600.slack.com/archives/CHM82GF99"},
			expectFn: func(sc *mockClienter) {
				sc.EXPECT().GetConversationHistoryContext(gomock.Any(), gomock.Any()).Return(
					&slack.GetConversationHistoryResponse{
						Messages:      []slack.Message{testMsg1.Message},
						SlackResponse: slack.SlackResponse{Ok: true},
					},
					nil,
				)
				mockConvInfo(sc, "CHM82GF99", "unittest")
			},
			want:    &types.Conversation{Name: "unittest", ID: "CHM82GF99", Messages: []types.Message{testMsg1}},
			wantErr: false,
		},
		{
			name:   "thread url",
			fields: fields{options: DefOptions},
			args:   args{context.Background(), "https://ora600.slack.com/archives/CHM82GF99/p1577694990000400"},
			expectFn: func(sc *mockClienter) {
				sc.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return(
					[]slack.Message{testMsg1.Message},
					false,
					"",
					nil,
				)
				mockConvInfo(sc, "CHM82GF99", "unittest")
			},
			want:    &types.Conversation{Name: "unittest", ID: "CHM82GF99", ThreadTS: "1577694990.000400", Messages: []types.Message{testMsg1}},
			wantErr: false,
		},
		{
			name:    "invalid url",
			fields:  fields{options: DefOptions},
			args:    args{context.Background(), "https://example.com"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := newmockClienter(ctrl)

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			sd := &Session{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.DumpAll(tt.args.ctx, tt.args.slackURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.DumpAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.DumpAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockConvInfo(mc *mockClienter, channelID, wantName string) {
	mc.EXPECT().GetConversationInfoContext(gomock.Any(), channelID, false).Return(&slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{NameNormalized: wantName}}}, nil)
}

func TestConversation_String(t *testing.T) {
	type fields struct {
		Messages []types.Message
		ID       string
		ThreadTS string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"conversation",
			fields{ID: "x"},
			"x",
		},
		{
			"thread",
			fields{ID: "x", ThreadTS: "y"},
			"x-y",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := types.Conversation{
				Messages: tt.fields.Messages,
				ID:       tt.fields.ID,
				ThreadTS: tt.fields.ThreadTS,
			}
			if got := c.String(); got != tt.want {
				t.Errorf("Conversation.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_getChannelName(t *testing.T) {
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		ctx       context.Context
		l         *rate.Limiter
		channelID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mockClienter)
		want     string
		wantErr  bool
	}{
		{
			name:   "ok",
			fields: fields{},
			args: args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				channelID: "TESTCHAN",
			},
			expectFn: func(sc *mockClienter) {
				sc.EXPECT().GetConversationInfoContext(gomock.Any(), "TESTCHAN", false).Return(&slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{NameNormalized: "unittest"}}}, nil)
			},
			want:    "unittest",
			wantErr: false,
		},
		{
			name:   "error",
			fields: fields{},
			args: args{
				ctx:       context.Background(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				channelID: "TESTCHAN",
			},
			expectFn: func(sc *mockClienter) {
				sc.EXPECT().GetConversationInfoContext(gomock.Any(), "TESTCHAN", false).Return(nil, errors.New("rekt"))
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := newmockClienter(ctrl)

			tt.expectFn(mc)
			sd := &Session{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.getChannelName(tt.args.ctx, tt.args.l, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.getChannelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Session.getChannelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsBotMessage(t *testing.T) {
	tests := []struct {
		name string
		m    types.Message
		want bool
	}{
		{"not a bot",
			fixtures.Load[types.Message](fixtures.ThreadMessage1JSON),
			false,
		},
		{"bot message",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsBotMessage(); got != tt.want {
				t.Errorf("Message.IsBotMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsThread(t *testing.T) {
	tests := []struct {
		name string
		m    types.Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{"not a thread",
			fixtures.Load[types.Message](fixtures.SimpleMessageJSON),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsThread(); got != tt.want {
				t.Errorf("Message.IsThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsThreadParent(t *testing.T) {
	tests := []struct {
		name string
		m    types.Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			false,
		},
		{"not a thread",
			fixtures.Load[types.Message](fixtures.SimpleMessageJSON),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsThreadParent(); got != tt.want {
				t.Errorf("Message.IsThreadParent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsThreadChild(t *testing.T) {
	tests := []struct {
		name string
		m    types.Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			false,
		},
		{"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{"not a thread",
			fixtures.Load[types.Message](fixtures.SimpleMessageJSON),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsThreadChild(); got != tt.want {
				t.Errorf("Message.IsThreadChild() = %v, want %v", got, tt.want)
			}
		})
	}
}
