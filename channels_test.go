package slackdump

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

func TestSlackDumper_getChannels(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
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
		want     Channels
		wantErr  bool
	}{
		{
			"ok",
			fields{},
			args{
				context.Background(),
				allChanTypes,
			},
			func(mc *mockClienter) {
				mc.EXPECT().GetConversationsContext(context.Background(), &slack.GetConversationsParameters{
					Types: allChanTypes,
				}).Return(Channels{
					slack.Channel{GroupConversation: slack.GroupConversation{
						Name: "lol",
					}}},
					"",
					nil)
			},
			Channels{slack.Channel{GroupConversation: slack.GroupConversation{
				Name: "lol",
			}}},
			false,
		},
		{
			"function made a boo boo",
			fields{},
			args{
				context.Background(),
				allChanTypes,
			},
			func(mc *mockClienter) {
				mc.EXPECT().GetConversationsContext(context.Background(), &slack.GetConversationsParameters{
					Types: allChanTypes,
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
			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}

			if tt.expectFn != nil {
				tt.expectFn(mc)
			}

			got, err := sd.getChannels(tt.args.ctx, tt.args.chanTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.getChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.getChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_GetChannels(t *testing.T) {
	type fields struct {
		client    clienter
		Users     Users
		UserIndex map[string]*slack.User
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
		want    Channels
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.GetChannels(tt.args.ctx, tt.args.chanTypes...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.GetChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.GetChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}
