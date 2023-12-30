package slackdump

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSession_getChannels(t *testing.T) {
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
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
			fields{options: DefOptions},
			args{
				context.Background(),
				AllChanTypes,
			},
			func(mc *mockClienter) {
				mc.EXPECT().GetConversationsContext(gomock.Any(), &slack.GetConversationsParameters{
					Limit: DefOptions.ChannelsPerReq,
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
			fields{options: DefOptions},
			args{
				context.Background(),
				AllChanTypes,
			},
			func(mc *mockClienter) {
				mc.EXPECT().GetConversationsContext(gomock.Any(), &slack.GetConversationsParameters{
					Limit: DefOptions.ChannelsPerReq,
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
			mc := newmockClienter(gomock.NewController(t))
			sd := &Session{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
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
		client    clienter
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
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
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
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
		options   Options
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
			fields{options: DefOptions},
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
			fields{options: DefOptions},
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
			fields{options: DefOptions},
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
			mc := newmockClienter(gomock.NewController(t))
			tt.expect(mc)
			sd := &Session{
				client:    mc,
				wspInfo:   tt.fields.wspInfo,
				fs:        tt.fields.fs,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
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
