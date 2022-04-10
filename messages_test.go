package slackdump

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/fixtures"
)

var (
	testMsg1 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "d1831c57-3b7f-4a0c-ab9a-a18d4a58a01c",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497751.040300",
		Text:        "Test message \u0026lt; \u0026gt; \u0026lt; \u0026gt;",
	}}}
	testMsg2 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "b11431d3-a5c4-4612-b09c-b074e9ddace7",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497781.040300",
		Text:        "message 2",
	}}}
	testMsg3 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "a99df2f2-1fd6-421f-9453-6903974b683a",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1641541791.000000",
		Text:        "message 3",
	}}}
	testMsg4t = Message{
		Message: slack.Message{Msg: slack.Msg{
			ClientMsgID:     "931db474-6ea8-43bc-9ff7-804309716ded",
			Type:            "message",
			User:            "UP58RAHCJ",
			Timestamp:       "1638524854.042000",
			ThreadTimestamp: "1638524854.042000",
			ReplyCount:      3,
			Text:            "message 4",
		}},
		ThreadReplies: []Message{
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
				testMsg1.Message,
				testMsg2.Message,
				testMsg3.Message,
			}},
			[]Message{
				testMsg1,
				testMsg2,
				testMsg3,
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
		want     *Conversation
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
			&Conversation{
				Name: "channel_name",
				ID:   "CHANNEL",
				Messages: []Message{
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
			&Conversation{
				Name: "channel_name",
				ID:   "CHANNEL",
				Messages: []Message{
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

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.DumpAllMessages(tt.args.ctx, tt.args.channelID)
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
			args{context.Background(), newLimiter(noTier, 1, 0), "CHANNEL", "THREAD"},
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
			args{context.Background(), newLimiter(noTier, 1, 0), "CHANNEL", "THREAD"},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
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
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah"},
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
				l:         newLimiter(noTier, 1, 0),
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
				l:         newLimiter(noTier, 1, 0),
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
				l:         newLimiter(noTier, 1, 0),
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

func TestSlackDumper_generateText(t *testing.T) {
	type fields struct {
		client    clienter
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		m      []Message
		prefix string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"two messages from the same person, not very far apart, with html escaped char",
			fields{},
			args{[]Message{testMsg1, testMsg2}, ""},
			"\n> U10H7D9RR [U10H7D9RR] @ 03/12/2021 02:15:51 Z:\nTest message < > < >\nmessage 2\n",
			false,
		},
		{
			"two messages from the same person, far apart",
			fields{},
			args{[]Message{testMsg1, testMsg4t}, ""},
			"\n> U10H7D9RR [U10H7D9RR] @ 03/12/2021 02:15:51 Z:\nTest message < > < >\n\n> UP58RAHCJ [UP58RAHCJ] @ 03/12/2021 09:47:34 Z:\nmessage 4\n|   \n|   > U01HPAR0YFN [U01HPAR0YFN] @ 03/12/2021 18:05:26 Z:\n|   blah blah, reply 1\n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			w := &bytes.Buffer{}
			if err := sd.generateText(w, tt.args.m, tt.args.prefix); (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.generateText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotW := w.String()
			assert.Equal(t, tt.wantW, gotW)
		})
	}
}

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
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
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
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD"},
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
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah"},
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

func TestSlackDumper_DumpURL(t *testing.T) {
	t.Parallel()
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
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
		want     *Conversation
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
			want:    &Conversation{Name: "unittest", ID: "CHM82GF99", Messages: []Message{testMsg1}},
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
			want:    &Conversation{Name: "unittest", ID: "CHM82GF99", ThreadTS: "1577694990.000400", Messages: []Message{testMsg1}},
			wantErr: false,
		},
		{
			name:    "invalid url",
			fields:  fields{options: DefOptions},
			args:    args{context.Background(), "x"},
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

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.DumpAllURL(tt.args.ctx, tt.args.slackURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.DumpURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.DumpURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mockConvInfo(mc *mockClienter, channelID, wantName string) {
	mc.EXPECT().GetConversationInfoContext(gomock.Any(), channelID, false).Return(&slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{NameNormalized: wantName}}}, nil)
}

func TestConversation_String(t *testing.T) {
	type fields struct {
		Messages []Message
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
			c := Conversation{
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

func TestSlackDumper_convHistoryParams(t *testing.T) {
	type fields struct {
		client    clienter
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		channelID string
		cursor    string
		oldest    time.Time
		latest    time.Time
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *slack.GetConversationHistoryParameters
	}{
		{
			"just channel",
			fields{options: DefOptions},
			args{channelID: "CHAN_ID"},
			&slack.GetConversationHistoryParameters{
				ChannelID: "CHAN_ID",
				Limit:     DefOptions.ConversationsPerReq,
			},
		},
		{
			"channel and cursor",
			fields{options: DefOptions},
			args{channelID: "CHAN_ID", cursor: "SOME_JUNK"},
			&slack.GetConversationHistoryParameters{
				ChannelID: "CHAN_ID",
				Cursor:    "SOME_JUNK",
				Limit:     DefOptions.ConversationsPerReq,
			},
		},
		{
			"oldest set",
			fields{options: DefOptions},
			args{channelID: "CHAN_ID", oldest: time.Date(1991, 9, 16, 6, 7, 8, 9, time.UTC)},
			&slack.GetConversationHistoryParameters{
				ChannelID: "CHAN_ID",
				Oldest:    "685001228.000009",
				Inclusive: true,
				Limit:     DefOptions.ConversationsPerReq,
			},
		},
		{
			"latest set",
			fields{options: DefOptions},
			args{channelID: "CHAN_ID", latest: time.Date(2020, 1, 5, 6, 7, 8, 9, time.UTC)},
			&slack.GetConversationHistoryParameters{
				ChannelID: "CHAN_ID",
				Latest:    "1578204428.000009",
				Inclusive: true,
				Limit:     DefOptions.ConversationsPerReq,
			},
		},
		{
			"full house",
			fields{options: DefOptions},
			args{
				channelID: "CHAN_ID",
				cursor:    "JUNK",
				oldest:    time.Date(1991, 9, 16, 6, 7, 8, 9, time.UTC),
				latest:    time.Date(2020, 1, 5, 6, 7, 8, 9, time.UTC),
			},
			&slack.GetConversationHistoryParameters{
				ChannelID: "CHAN_ID",
				Cursor:    "JUNK",
				Inclusive: true,
				Oldest:    "685001228.000009",
				Latest:    "1578204428.000009",
				Limit:     DefOptions.ConversationsPerReq,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			if got := sd.convHistoryParams(tt.args.channelID, tt.args.cursor, tt.args.oldest, tt.args.latest); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.convHistoryParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_getChannelName(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
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
				l:         newLimiter(noTier, 1, 0),
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
				l:         newLimiter(noTier, 1, 0),
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
			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.getChannelName(tt.args.ctx, tt.args.l, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.getChannelName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackDumper.getChannelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessage_IsBotMessage(t *testing.T) {
	tests := []struct {
		name string
		m    Message
		want bool
	}{
		{"not a bot",
			fixtures.Load[Message](fixtures.ThreadMessage1JSON),
			false,
		},
		{"bot message",
			fixtures.Load[Message](fixtures.BotMessageThreadParentJSON),
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
		m    Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{"is thread (child)",
			fixtures.Load[Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{"not a thread",
			fixtures.Load[Message](fixtures.SimpleMessageJSON),
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
		m    Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[Message](fixtures.BotMessageThreadParentJSON),
			true,
		},
		{"is thread (child)",
			fixtures.Load[Message](fixtures.BotMessageThreadChildJSON),
			false,
		},
		{"not a thread",
			fixtures.Load[Message](fixtures.SimpleMessageJSON),
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
		m    Message
		want bool
	}{
		{"is thread (parent)",
			fixtures.Load[Message](fixtures.BotMessageThreadParentJSON),
			false,
		},
		{"is thread (child)",
			fixtures.Load[Message](fixtures.BotMessageThreadChildJSON),
			true,
		},
		{"not a thread",
			fixtures.Load[Message](fixtures.SimpleMessageJSON),
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
