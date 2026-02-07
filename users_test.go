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
	"time"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/client/mock_client"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

const testSuffix = "UNIT"

var testUsers = types.Users(fixtures.TestUsers)

func TestUsers_IndexByID(t *testing.T) {
	users := []slack.User{
		{ID: "USLACKBOT", Name: "slackbot"},
		{ID: "USER2", Name: "User 2"},
	}
	tests := []struct {
		name string
		us   types.Users
		want structures.UserIndex
	}{
		{"test 1", users, structures.UserIndex{
			"USLACKBOT": &users[0],
			"USER2":     &users[1],
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.us.IndexByID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Users.MakeUserIDIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_fetchUsers(t *testing.T) {
	type fields struct {
		Users  types.Users
		config config
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_client.MockSlackClienter)
		want     types.Users
		wantErr  bool
	}{
		{
			"ok",
			fields{config: defConfig},
			args{t.Context()},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User(testUsers), nil)
			},
			testUsers,
			false,
		},
		{
			"api error",
			fields{config: defConfig},
			args{t.Context()},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return(nil, errors.New("i don't think so"))
			},
			nil,
			true,
		},
		{
			"zero users",
			fields{config: defConfig},
			args{t.Context()},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User{}, nil)
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mock_client.NewMockSlackClienter(gomock.NewController(t))

			tt.expectFn(mc)

			sd := &Session{
				client: mc,
				cfg:    tt.fields.config,
			}
			got, err := sd.fetchUsers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.fetchUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.fetchUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_GetUsers(t *testing.T) {
	type fields struct {
		config    config
		usercache usercache
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_client.MockSlackClienter)
		want     types.Users
		wantErr  bool
	}{
		{
			"everything goes as planned",
			fields{
				config: config{limits: network.Limits{
					Tier2: network.TierLimit{Burst: 1},
					Tier3: network.TierLimit{Burst: 1},
				}},
				usercache: usercache{},
			},
			args{t.Context()},
			func(mc *mock_client.MockSlackClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User(testUsers), nil)
			},
			testUsers,
			false,
		},
		{
			"loaded from cache",
			fields{
				config: config{limits: network.Limits{
					Tier2: network.TierLimit{Burst: 1},
					Tier3: network.TierLimit{Burst: 1},
				}},
				usercache: usercache{
					users:    testUsers,
					cachedAt: time.Now(),
				},
			},
			args{t.Context()},
			func(mc *mock_client.MockSlackClienter) {
				// we don't expect any API calls
			},
			testUsers,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := mock_client.NewMockSlackClienter(gomock.NewController(t))

			tt.expectFn(mc)

			sd := &Session{
				client:  mc,
				wspInfo: &slack.AuthTestResponse{TeamID: testSuffix},
				cfg:     tt.fields.config,
				uc:      &tt.fields.usercache,
				log:     slog.Default(),
			}
			got, err := sd.GetUsers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.GetUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.GetUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}
