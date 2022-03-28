package slackdump

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

const testSuffix = "UNIT"

var testUsers = Users{
	{ID: "LOL1", Name: "yippi", Deleted: false},
	{ID: "DELD", Name: "ka", Deleted: true},
	{ID: "LOL3", Name: "yay", Deleted: false},
	{ID: "LOL4", Name: "motherfucker", Deleted: false, IsBot: true},
}

func TestUsers_IndexByID(t *testing.T) {
	users := []slack.User{
		{ID: "USLACKBOT", Name: "slackbot"},
		{ID: "USER2", Name: "User 2"},
	}
	tests := []struct {
		name string
		us   Users
		want map[string]*slack.User
	}{
		{"test 1", users, map[string]*slack.User{
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

func TestSlackDumper_IsUserDeleted(t *testing.T) {
	type fields struct {
		client    *slack.Client
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "deleted",
			fields: fields{
				Users:     testUsers,
				UserIndex: testUsers.IndexByID(),
			},
			args: args{"DELD"},
			want: true,
		},
		{
			name: "not deleted",
			fields: fields{
				Users:     testUsers,
				UserIndex: testUsers.IndexByID(),
			},
			args: args{"LOL1"},
			want: false,
		},
		{
			name: "not present",
			fields: fields{
				Users:     testUsers,
				UserIndex: testUsers.IndexByID(),
			},
			args: args{"XXX"},
			want: false,
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
			if got := sd.IsUserDeleted(tt.args.id); got != tt.want {
				t.Errorf("SlackDumper.IsDeletedUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsers_ToText(t *testing.T) {
	type args struct {
		sd *SlackDumper
	}
	tests := []struct {
		name    string
		us      Users
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"test user list",
			testUsers,
			args{nil},
			"Name          ID    Bot?  Deleted?\n                          \nka            DELD        deleted\nmotherfucker  LOL4  bot   \nyay           LOL3        \nyippi         LOL1        \n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.us.ToText(tt.args.sd, w); (err != nil) != tt.wantErr {
				t.Errorf("Users.ToText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Users.ToText() = %q, want %q", gotW, tt.wantW)
			}
		})
	}
}

func TestSlackDumper_saveUserCache(t *testing.T) {

	// test saving file works
	sd := SlackDumper{teamID: testSuffix}

	dir := t.TempDir()
	testfile := filepath.Join(dir, "test.json")

	assert.NoError(t, sd.saveUserCache(testfile, testSuffix, testUsers))

	reopenedF, err := os.Open(makeCacheFilename(testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	var uu Users
	assert.NoError(t, json.NewDecoder(reopenedF).Decode(&uu))
	assert.Equal(t, testUsers, uu)
}

func TestSlackDumper_loadUserCache(t *testing.T) {
	type fields struct {
		client    *slack.Client
		Users     Users
		UserIndex map[string]*slack.User
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
		want    Users
		wantErr bool
	}{
		{
			"loads the cache ok",
			fields{},
			args{gimmeTempFileWithUsers(t), 5 * time.Hour},
			testUsers,
			false,
		},
		{
			"no data",
			fields{},
			args{gimmeTempFile(t), 5 * time.Hour},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Remove(tt.args.filename)
			sd := &SlackDumper{
				client:    tt.fields.client,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.loadUserCache(tt.args.filename, testSuffix, tt.args.maxAge)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.loadUserCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.loadUserCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_fetchUsers(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
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
		want     Users
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

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.fetchUsers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.fetchUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.fetchUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackDumper_GetUsers(t *testing.T) {
	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
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
		want     Users
		wantErr  bool
	}{
		{
			"everything goes as planned",
			fields{options: Options{
				UserCacheFilename: gimmeTempFile(t),
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
				UserCacheFilename: gimmeTempFileWithUsers(t),
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
			defer os.Remove(tt.fields.options.UserCacheFilename)

			mc := newmockClienter(gomock.NewController(t))

			tt.expectFn(mc)

			sd := &SlackDumper{
				client:    mc,
				teamID:    testSuffix,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.GetUsers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.GetUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.GetUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func gimmeTempFile(t *testing.T) string {
	f, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func gimmeTempFileWithUsers(t *testing.T) string {
	f := gimmeTempFile(t)
	sd := SlackDumper{}
	if err := sd.saveUserCache(f, testSuffix, testUsers); err != nil {
		t.Fatal(err)
	}
	return f
}
