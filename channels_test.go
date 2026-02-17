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

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/client/mock_client"
	"github.com/rusq/slackdump/v4/internal/edge"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/stream"
	"github.com/rusq/slackdump/v4/types"
)

func TestSession_getChannels(t *testing.T) {
	type fields struct {
		config config
	}
	type args struct {
		ctx       context.Context
		chanTypes []string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mock_client.MockSlackClienter)
		want     types.Channels
		wantErr  bool
	}{
		{
			"ok",
			fields{config: defConfig},
			args{
				t.Context(),
				AllChanTypes,
			},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetConversationsContext(gomock.Any(), &slack.GetConversationsParameters{
					Limit: network.DefLimits.Request.Channels,
					Types: AllChanTypes,
				}).Return(types.Channels{
					slack.Channel{GroupConversation: slack.GroupConversation{
						Name: "lol",
					}},
				},
					"",
					nil)
			},
			types.Channels{slack.Channel{GroupConversation: slack.GroupConversation{
				Name: "lol",
			}}},
			false,
		},
		{
			"function made a boo boo",
			fields{config: defConfig},
			args{
				t.Context(),
				AllChanTypes,
			},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetConversationsContext(gomock.Any(), &slack.GetConversationsParameters{
					Limit: network.DefLimits.Request.Channels,
					Types: AllChanTypes,
				}).Return(
					nil,
					"",
					errors.New("boo boo"))
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mock_client.NewMockSlackClienter(gomock.NewController(t))
			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
			}

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			var got types.Channels
			err := sd.getChannels(tt.args.ctx, GetChannelsParameters{ChannelTypes: tt.args.chanTypes}, func(_ context.Context, c types.Channels) error {
				got = append(got, c...)
				return nil
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.getChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSession_GetChannels(t *testing.T) {
	type fields struct {
		client client.SlackClienter
		config config
	}
	type args struct {
		ctx       context.Context
		chanTypes []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.Channels
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &Session{
				client: tt.fields.client,
				cfg:    tt.fields.config,
			}
			got, err := sd.GetChannels(tt.args.ctx, tt.args.chanTypes...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.GetChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.GetChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldFallbackToListChannels(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "op not supported",
			err:  stream.ErrOpNotSupported,
			want: true,
		},
		{
			name: "api no_channels_supplied",
			err:  &edge.APIError{Err: "no_channels_supplied"},
			want: true,
		},
		{
			name: "api internal_error",
			err:  &edge.APIError{Err: "internal_error"},
			want: true,
		},
		{
			name: "wrapped callback no_channels_supplied",
			err:  errors.New("API error: callback error: no_channels_supplied"),
			want: true,
		},
		{
			name: "wrapped callback internal_error",
			err:  errors.New("API error: callback error: internal_error"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("network timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldFallbackToListChannels(tt.err); got != tt.want {
				t.Fatalf("shouldFallbackToListChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_GetChannelMembers(t *testing.T) {
	type fields struct {
		wspInfo   *slack.AuthTestResponse
		fs        fsadapter.FS
		Users     types.Users
		UserIndex structures.UserIndex
		cfg       config
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expect  func(mc *mock_client.MockSlackClienter)
		want    []string
		wantErr bool
	}{
		{
			"ok, single call",
			fields{cfg: defConfig},
			args{
				t.Context(),
				"chanID",
			},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersInConversationContext(gomock.Any(), &slack.GetUsersInConversationParameters{
					ChannelID: "chanID",
				}).Return([]string{"user1", "user2"}, "", nil)
			},
			[]string{"user1", "user2"},
			false,
		},
		{
			"ok, two calls",
			fields{cfg: defConfig},
			args{
				t.Context(),
				"chanID",
			},
			func(mc *mock_client.MockSlackClienter) {
				first := mc.EXPECT().GetUsersInConversationContext(gomock.Any(), &slack.GetUsersInConversationParameters{
					ChannelID: "chanID",
				}).Return([]string{"user1", "user2"}, "cursor", nil).Times(1)
				_ = mc.EXPECT().GetUsersInConversationContext(gomock.Any(), &slack.GetUsersInConversationParameters{
					ChannelID: "chanID",
					Cursor:    "cursor",
				}).Return([]string{"user3"}, "", nil).After(first).Times(1)
			},
			[]string{"user1", "user2", "user3"},
			false,
		},
		{
			"error",
			fields{cfg: defConfig},
			args{
				t.Context(),
				"chanID",
			},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersInConversationContext(gomock.Any(), &slack.GetUsersInConversationParameters{
					ChannelID: "chanID",
				}).Return([]string{}, "", errors.New("error fornicating corrugations"))
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mock_client.NewMockSlackClienter(gomock.NewController(t))
			tt.expect(mc)
			sd := &Session{
				client:  mc,
				wspInfo: tt.fields.wspInfo,
				fs:      tt.fields.fs,
				cfg:     tt.fields.cfg,
			}
			got, err := sd.GetChannelMembers(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.GetChannelMembers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.GetChannelMembers() = %v, want %v", got, tt.want)
			}
		})
	}
}
