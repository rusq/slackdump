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
package list

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/types"
)

func Test_getCachedUsers(t *testing.T) {
	var (
		testUsers = []slack.User{
			{ID: "U1"},
			{ID: "U2"},
			{ID: "U3"},
		}
	)
	type args struct {
		ctx       context.Context
		skipCache bool
		teamID    string
	}
	tests := []struct {
		name    string
		args    args
		expect  func(c *MockuserCacher, g *MockuserGetter)
		want    []slack.User
		wantErr bool
	}{
		/* oh happy days */
		{
			"users loaded from cache",
			args{t.Context(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(testUsers, nil)
			},
			testUsers,
			false,
		},
		{
			"getting users from API ok (recoverable cache error)",
			args{t.Context(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(nil, &fs.PathError{})
				g.EXPECT().GetUsers(gomock.Any()).Return(testUsers, nil)
				c.EXPECT().CacheUsers("TEAM1", testUsers).Return(nil)
			},
			testUsers,
			false,
		},
		{
			"saving cache fails, but we continue",
			args{t.Context(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(nil, &fs.PathError{})
				g.EXPECT().GetUsers(gomock.Any()).Return(testUsers, nil)
				c.EXPECT().CacheUsers("TEAM1", testUsers).Return(errors.New("disk mulching detected"))
			},
			testUsers,
			false,
		},
		/* unhappy days */
		{
			"unrecoverable error",
			args{t.Context(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(nil, errors.New("frobnication error"))
			},
			nil,
			true,
		},
		{
			"getting users from API fails",
			args{t.Context(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(nil, &fs.PathError{})
				g.EXPECT().GetUsers(gomock.Any()).Return(nil, errors.New("blip"))
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			muc := NewMockuserCacher(ctrl)
			mug := NewMockuserGetter(ctrl)

			tt.expect(muc, mug)

			got, err := fetchUsers(tt.args.ctx, mug, muc, tt.args.skipCache, tt.args.teamID)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCachedUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCachedUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_users_Len(t *testing.T) {
	type fields struct {
		data   types.Users
		common commonOpts
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "zero",
			want: 0,
		},
		{
			name:   "three",
			fields: fields{data: make(types.Users, 3)},
			want:   3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &users{
				data:   tt.fields.data,
				common: tt.fields.common,
			}
			if got := u.Len(); got != tt.want {
				t.Errorf("users.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}
