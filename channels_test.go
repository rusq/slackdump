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

	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
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
		expectFn func(mc *mockClienter)
		want     types.Channels
		wantErr  bool
	}{
		{
			"ok",
			fields{config: defConfig},
			args{
				context.Background(),
				AllChanTypes,
			},
			func(mc *mockClienter) {
				mc.EXPECT().GetConversationsContext(gomock.Any(), &slack.GetConversationsParameters{
					Limit: network.DefLimits.Request.Channels,
					Types: AllChanTypes,
				}).Return(types.Channels{
					slack.Channel{GroupConversation: slack.GroupConversation{
						Name: "lol",
					}}},
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
				context.Background(),
				AllChanTypes,
			},
			func(mc *mockClienter) {
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
			mc := NewmockClienter(gomock.NewController(t))
			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
				log:    slog.Default(),
			}

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			var got types.Channels
			err := sd.getChannels(tt.args.ctx, tt.args.chanTypes, func(c types.Channels) error {
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
		client clienter
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
		expect  func(mc *mockClienter)
		want    []string
		wantErr bool
	}{
		{
			"ok, single call",
			fields{cfg: defConfig},
			args{
				context.Background(),
				"chanID",
			},
			func(mc *mockClienter) {
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
				context.Background(),
				"chanID",
			},
			func(mc *mockClienter) {
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
				context.Background(),
				"chanID",
			},
			func(mc *mockClienter) {
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
			mc := NewmockClienter(gomock.NewController(t))
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
