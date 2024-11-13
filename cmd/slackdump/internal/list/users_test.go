package list

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"
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
			args{context.Background(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(testUsers, nil)
			},
			testUsers,
			false,
		},
		{
			"getting users from API ok (recoverable cache error)",
			args{context.Background(), false, "TEAM1"},
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
			args{context.Background(), false, "TEAM1"},
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
			args{context.Background(), false, "TEAM1"},
			func(c *MockuserCacher, g *MockuserGetter) {
				c.EXPECT().LoadUsers("TEAM1", gomock.Any()).Return(nil, errors.New("frobnication error"))
			},
			nil,
			true,
		},
		{
			"getting users from API fails",
			args{context.Background(), false, "TEAM1"},
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
