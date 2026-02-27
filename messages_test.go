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

package slackdump

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v4/internal/client/mock_client"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/types"
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
		config config
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mock_client.MockSlack)
		want     *types.Conversation
		wantErr  bool
	}{
		{
			"all ok",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL"},
			func(c *mock_client.MockSlack) {
				c.EXPECT().GetConversationHistoryContext(
					gomock.Any(),
					&slack.GetConversationHistoryParameters{
						ChannelID: "CHANNEL",
						Limit:     network.DefLimits.Request.Conversations,
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
				},
			},
			false,
		},
		{
			"channelID is empty",
			fields{config: defConfig},
			args{t.Context(), ""},
			func(c *mock_client.MockSlack) {},
			nil,
			true,
		},
		{
			"iteration test",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL"},
			func(c *mock_client.MockSlack) {
				first := c.EXPECT().
					GetConversationHistoryContext(
						gomock.Any(),
						&slack.GetConversationHistoryParameters{
							ChannelID: "CHANNEL",
							Limit:     network.DefLimits.Request.Conversations,
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
							Limit:     network.DefLimits.Request.Conversations,
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
				},
			},
			false,
		},
		{
			"resp not ok",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL"},
			func(c *mock_client.MockSlack) {
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
			fields{config: defConfig},
			args{t.Context(), "CHANNEL"},
			func(c *mock_client.MockSlack) {
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
			mc := mock_client.NewMockSlack(ctrl)

			tt.expectFn(mc)

			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
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
		config config
	}
	type args struct {
		ctx      context.Context
		slackURL string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(sc *mock_client.MockSlack)
		want     *types.Conversation
		wantErr  bool
	}{
		{
			name:   "conversation url",
			fields: fields{config: defConfig},
			args:   args{t.Context(), "https://ora600.slack.com/archives/CHM82GF99"},
			expectFn: func(sc *mock_client.MockSlack) {
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
			fields: fields{config: defConfig},
			args:   args{t.Context(), "https://ora600.slack.com/archives/CHM82GF99/p1577694990000400"},
			expectFn: func(sc *mock_client.MockSlack) {
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
			fields:  fields{config: defConfig},
			args:    args{t.Context(), "https://example.com"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_client.NewMockSlack(ctrl)

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
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

func mockConvInfo(mc *mock_client.MockSlack, channelID, wantName string) {
	mc.EXPECT().
		GetConversationInfoContext(gomock.Any(), &slack.GetConversationInfoInput{ChannelID: channelID}).
		Return(&slack.Channel{GroupConversation: slack.GroupConversation{Name: wantName, Conversation: slack.Conversation{NameNormalized: wantName + "_normalized"}}}, nil)
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
		config config
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
		expectFn func(mc *mock_client.MockSlack)
		want     string
		wantErr  bool
	}{
		{
			name:   "ok",
			fields: fields{},
			args: args{
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				channelID: "TESTCHAN",
			},
			expectFn: func(sc *mock_client.MockSlack) {
				sc.EXPECT().
					GetConversationInfoContext(gomock.Any(), &slack.GetConversationInfoInput{ChannelID: "TESTCHAN"}).
					Return(&slack.Channel{GroupConversation: slack.GroupConversation{Name: "unittest", Conversation: slack.Conversation{NameNormalized: "unittest_normalized"}}}, nil)
			},
			want:    "unittest",
			wantErr: false,
		},
		{
			name:   "error",
			fields: fields{},
			args: args{
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				channelID: "TESTCHAN",
			},
			expectFn: func(sc *mock_client.MockSlack) {
				sc.EXPECT().
					GetConversationInfoContext(gomock.Any(), &slack.GetConversationInfoInput{ChannelID: "TESTCHAN"}).
					Return(nil, errors.New("rekt"))
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_client.NewMockSlack(ctrl)

			tt.expectFn(mc)
			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
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
		{
			"not a bot",
			fixtures.Load[types.Message](fixtures.ThreadMessage1JSON),
			false,
		},
		{
			"bot message",
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
		{
			"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{
			"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{
			"not a thread",
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
		{
			"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{
			"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			false,
		},
		{
			"not a thread",
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
		{
			"is thread (parent)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadParentJSON),
			false,
		},
		{
			"is thread (child)",
			fixtures.Load[types.Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{
			"not a thread",
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
