package slackdump

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"errors"

	"github.com/golang/mock/gomock"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
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

func TestSession_saveUserCache(t *testing.T) {

	// test saving file works
	sd := Session{wspInfo: &slack.AuthTestResponse{TeamID: "123"}}

	dir := t.TempDir()
	testfile := filepath.Join(dir, "test.json")

	assert.NoError(t, sd.saveUserCache(testfile, testSuffix, testUsers))

	reopenedF, err := os.Open(sd.makeCacheFilename(testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	var uu types.Users
	assert.NoError(t, json.NewDecoder(reopenedF).Decode(&uu))
	assert.Equal(t, testUsers, uu)
}

func TestSession_loadUserCache(t *testing.T) {
	dir := t.TempDir()
	type fields struct {
		client    *slack.Client
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		filename string
		maxAge   time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    types.Users
		wantErr bool
	}{
		{
			"loads the cache ok",
			fields{},
			args{gimmeTempFileWithUsers(t, dir), 5 * time.Hour},
			testUsers,
			false,
		},
		{
			"no data",
			fields{},
			args{gimmeTempFile(t, dir), 5 * time.Hour},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &Session{
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.loadUserCache(tt.args.filename, testSuffix, tt.args.maxAge)
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.loadUserCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.loadUserCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSession_fetchUsers(t *testing.T) {
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mockClienter)
		want     types.Users
		wantErr  bool
	}{
		{
			"ok",
			fields{options: DefOptions},
			args{context.Background()},
			func(mc *mockClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User(testUsers), nil)
			},
			testUsers,
			false,
		},
		{
			"api error",
			fields{options: DefOptions},
			args{context.Background()},
			func(mc *mockClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return(nil, errors.New("i don't think so"))
			},
			nil,
			true,
		},
		{
			"zero users",
			fields{options: DefOptions},
			args{context.Background()},
			func(mc *mockClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User{}, nil)
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newmockClienter(gomock.NewController(t))

			tt.expectFn(mc)

			sd := &Session{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
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
	dir := t.TempDir()
	type fields struct {
		Users     types.Users
		UserIndex structures.UserIndex
		options   Options
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mockClienter)
		want     types.Users
		wantErr  bool
	}{
		{
			"everything goes as planned",
			fields{options: Options{
				UserCacheFilename: gimmeTempFile(t, dir),
				MaxUserCacheAge:   5 * time.Hour,
				Tier2Burst:        1,
				Tier3Burst:        1,
			}},
			args{context.Background()},
			func(mc *mockClienter) {
				mc.EXPECT().GetUsersContext(gomock.Any()).Return([]slack.User(testUsers), nil)
			},
			testUsers,
			false,
		},
		{
			"loaded from cache",
			fields{options: Options{
				UserCacheFilename: gimmeTempFileWithUsers(t, dir),
				MaxUserCacheAge:   5 * time.Hour,
				Tier2Burst:        1,
				Tier3Burst:        1,
			}},
			args{context.Background()},
			func(mc *mockClienter) {
				// we don't expect any API calls
			},
			testUsers,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newmockClienter(gomock.NewController(t))

			tt.expectFn(mc)

			sd := &Session{
				client:    mc,
				wspInfo:   &slack.AuthTestResponse{TeamID: testSuffix},
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
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

func gimmeTempFile(t *testing.T, dir string) string {
	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("error closing test file: %s", err)
	}
	return f.Name()
}

func gimmeTempFileWithUsers(t *testing.T, dir string) string {
	f := gimmeTempFile(t, dir)
	sd := Session{}
	if err := sd.saveUserCache(f, testSuffix, testUsers); err != nil {
		t.Fatal(err)
	}
	return f
}

func FuzzFilenameSplit(f *testing.F) {
	testInput := []string{
		"users.json",
		"channels.json",
	}
	for _, ti := range testInput {
		f.Add(ti)
	}
	f.Fuzz(func(t *testing.T, input string) {
		split := filenameSplit(input)
		joined := filenameJoin(split)
		assert.Equal(t, input, joined)
	})
}
