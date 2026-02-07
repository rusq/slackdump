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
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/internal/client/mock_client"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

func TestSession_DumpThreadWithFiles(t *testing.T) {
	type fields struct {
		Users  types.Users
		config config
	}
	type args struct {
		ctx       context.Context
		channelID string
		threadTS  string
		oldest    time.Time
		latest    time.Time
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_client.MockSlackClienter)
		want     *types.Conversation
		wantErr  bool
	}{
		{"chan and thread are empty", fields{config: defConfig}, args{t.Context(), "", "", time.Time{}, time.Time{}}, nil, nil, true},
		{"thread empty", fields{config: defConfig}, args{t.Context(), "xxx", "", time.Time{}, time.Time{}}, nil, nil, true},
		{"chan empty", fields{config: defConfig}, args{t.Context(), "", "yyy", time.Time{}, time.Time{}}, nil, nil, true},
		{
			"ok",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL", "THREAD", time.Time{}, time.Time{}},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Limit: network.DefLimits.Request.Replies, Inclusive: true},
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
			&types.Conversation{Name: "channel_name", ID: "CHANNEL", ThreadTS: "THREAD", Messages: []types.Message{testMsg1, testMsg2, testMsg3}},
			false,
		},
		{
			"ok with time constraints",
			fields{config: defConfig},
			args{
				t.Context(),
				"CHANNEL",
				"THREAD",
				time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC),
				time.Date(2021, 12, 31, 23, 59, 59, 0, time.UTC),
			},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{
							ChannelID: "CHANNEL",
							Timestamp: "THREAD",
							Limit:     network.DefLimits.Request.Replies,
							Oldest:    "1609459199.000000",
							Latest:    "1640995199.000000",
							Inclusive: true,
						},
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
			&types.Conversation{Name: "channel_name", ID: "CHANNEL", ThreadTS: "THREAD", Messages: []types.Message{testMsg1, testMsg2, testMsg3}},
			false,
		},
		{
			"iterating over",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL", "THREAD", time.Time{}, time.Time{}},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Limit: network.DefLimits.Request.Replies, Inclusive: true},
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
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Cursor: "blah", Limit: network.DefLimits.Request.Replies, Inclusive: true},
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
			&types.Conversation{Name: "channel_name", ID: "CHANNEL", ThreadTS: "THREAD", Messages: []types.Message{testMsg1, testMsg2}},
			false,
		},
		{
			"sudden bleep bloop error",
			fields{config: defConfig},
			args{t.Context(), "CHANNEL", "THREADTS", time.Time{}, time.Time{}},
			func(c *mock_client.MockSlackClienter) {
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
			mc := mock_client.NewMockSlackClienter(ctrl)

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
			}
			got, err := sd.dumpThreadAsConversation(tt.args.ctx, structures.SlackLink{Channel: tt.args.channelID, ThreadTS: tt.args.threadTS}, tt.args.oldest, tt.args.latest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.dumpThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSession_populateThreads(t *testing.T) {
	type args struct {
		ctx       context.Context
		l         *rate.Limiter
		msgs      []types.Message
		channelID string
		oldest    time.Time
		latest    time.Time
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
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []types.Message{testMsg1},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, oldest, latest time.Time, processFn ...ProcessFunc) ([]types.Message, error) {
					return nil, nil
				},
			},
			0,
			false,
		},
		{
			"ok, thread",
			args{
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []types.Message{testMsg1, testMsg4t},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, oldest, latest time.Time, processFn ...ProcessFunc) ([]types.Message, error) {
					return []types.Message{testMsg4t, testMsg2}, nil
				},
			},
			1,
			false,
		},
		{
			"skipping empty messages",
			args{
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []types.Message{testMsg4t, testMsg1},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, oldest, latest time.Time, processFn ...ProcessFunc) ([]types.Message, error) {
					return []types.Message{}, nil
				},
			},
			0,
			false,
		},
		{
			"failing on dumpFn returning error",
			args{
				ctx:       t.Context(),
				l:         network.NewLimiter(network.NoTier, 1, 0),
				msgs:      []types.Message{testMsg4t},
				channelID: "x",
				dumpFn: func(ctx context.Context, l *rate.Limiter, channelID, threadTS string, oldest, latest time.Time, processFn ...ProcessFunc) ([]types.Message, error) {
					return nil, errors.New("bam")
				},
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &Session{}
			got, err := sd.populateThreads(tt.args.ctx, tt.args.l, tt.args.msgs, tt.args.channelID, tt.args.oldest, tt.args.latest, tt.args.dumpFn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.populateThreads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Session.populateThreads() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_dumpThread(t *testing.T) {
	type fields struct {
		Users  types.Users
		config config
	}
	type args struct {
		ctx       context.Context
		l         *rate.Limiter
		channelID string
		threadTS  string
		oldest    time.Time
		latest    time.Time
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_client.MockSlackClienter)
		want     []types.Message
		wantErr  bool
	}{
		{
			"ok",
			fields{},
			args{t.Context(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREAD", time.Time{}, time.Time{}},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{ChannelID: "CHANNEL", Timestamp: "THREAD", Inclusive: true},
					).
					Return(
						[]slack.Message{testMsg1.Message, testMsg2.Message, testMsg3.Message},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]types.Message{testMsg1, testMsg2, testMsg3},
			false,
		},
		{
			"iterating over",
			fields{config: defConfig},
			args{t.Context(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREAD", time.Time{}, time.Time{}},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().
					GetConversationRepliesContext(
						gomock.Any(),
						&slack.GetConversationRepliesParameters{
							ChannelID: "CHANNEL",
							Timestamp: "THREAD",
							Limit:     network.DefLimits.Request.Replies,
							Inclusive: true,
						},
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
						&slack.GetConversationRepliesParameters{
							ChannelID: "CHANNEL",
							Cursor:    "blah",
							Timestamp: "THREAD",
							Limit:     network.DefLimits.Request.Replies,
							Inclusive: true,
						},
					).
					Return(
						[]slack.Message{testMsg2.Message},
						false,
						"",
						nil,
					).
					Times(1)
			},
			[]types.Message{testMsg1, testMsg2},
			false,
		},
		{
			"sudden bleep bloop error",
			fields{config: defConfig},
			args{t.Context(), network.NewLimiter(network.NoTier, 1, 0), "CHANNEL", "THREADTS", time.Time{}, time.Time{}},
			func(c *mock_client.MockSlackClienter) {
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
			mc := mock_client.NewMockSlackClienter(ctrl)

			tt.expectFn(mc)

			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
			}
			got, err := sd.dumpThread(tt.args.ctx, tt.args.l, tt.args.channelID, tt.args.threadTS, tt.args.oldest, tt.args.latest)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.dumpThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
