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
package stream

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime/trace"
	"testing"
	"time"

	"github.com/rusq/chttp"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/cache"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/client/mock_client"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
	"github.com/rusq/slackdump/v4/processor"
)

const testConversation = "CO720D65C25A"

func TestChannelStream(t *testing.T) {
	t.Skip()
	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	m, err := cache.NewManager(filepath.Join(ucd, "slackdump"))
	if err != nil {
		t.Fatal(err)
	}
	wsp, err := m.Current()
	if err != nil {
		t.Fatal(err)
	}
	prov, err := m.Auth(t.Context(), wsp, nil)
	if err != nil {
		t.Fatal(err)
	}

	sd := slack.New(prov.SlackToken(), slack.OptionHTTPClient(chttp.Must(chttp.New(auth.SlackURL, prov.Cookies()))))

	f, err := os.Create("record.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rec := chunk.NewRecorder(f)
	defer rec.Close()

	cs := New(sd, network.DefLimits)
	if err := cs.SyncConversations(t.Context(), rec, structures.EntityItem{Id: testConversation}); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderStream(t *testing.T) {
	ctx, task := trace.NewTask(t.Context(), "TestRecorderStream")
	defer task.End()

	start := time.Now()
	f := fixtures.ChunkFileJSONL()

	rgnNewSrv := trace.StartRegion(ctx, "NewServer")
	srv := chunktest.NewServer(f, "U123")
	rgnNewSrv.End()
	defer srv.Close()
	sd := slack.New("test", slack.OptionAPIURL(srv.URL()))

	w, err := os.Create(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	rec := chunk.NewRecorder(w)
	defer rec.Close()

	rgnStream := trace.StartRegion(ctx, "Stream")
	cs := New(sd, network.NoLimits)
	if err := cs.SyncConversations(ctx, rec, structures.EntityItem{Id: fixtures.ChunkFileChannelID}); err != nil {
		t.Fatal(err)
	}
	rgnStream.End()
	if time.Since(start) > 2*time.Second {
		t.Fatal("took too long")
	}
}

func TestReplay(t *testing.T) {
	f := fixtures.ChunkFileJSONL()
	srv := chunktest.NewServer(f, "U123")
	defer srv.Close()
	sd := slack.New("test", slack.OptionAPIURL(srv.URL()))

	reachedEnd := false
	for i := 0; i < 100_000; i++ {
		resp, err := sd.GetConversationHistory(&slack.GetConversationHistoryParameters{ChannelID: fixtures.ChunkFileChannelID})
		if err != nil {
			t.Fatalf("error on iteration %d: %s", i, err)
		}
		if !resp.HasMore {
			reachedEnd = true
			t.Log("no more messages")
			break
		}
	}
	if !reachedEnd {
		t.Fatal("didn't reach end of stream")
	}
}

var testThread = []slack.Message{
	{
		Msg: slack.Msg{
			Channel:         "CTM1",
			Timestamp:       "1610000000.000000",
			ThreadTimestamp: "1610000000.000000",
			Files: []slack.File{
				{ID: "FILE_1", Name: "file1"},
				{ID: "FILE_2", Name: "file2"},
			},
		},
	},
	{
		Msg: slack.Msg{
			Channel:         "CTM1",
			Timestamp:       "1610000000.000001",
			ThreadTimestamp: "1610000000.000000",
			Files: []slack.File{
				{ID: "FILE_3", Name: "file1"},
				{ID: "FILE_4", Name: "file2"},
			},
		},
	},
	{
		Msg: slack.Msg{
			Channel:         "CTM1",
			Timestamp:       "1610000000.000002",
			ThreadTimestamp: "1610000000.000000",
			Files: []slack.File{
				{ID: "FILE_5", Name: "file5"},
				{ID: "FILE_6", Name: "file6"},
			},
		},
	},
}

func Test_processThreadMessages(t *testing.T) {
	t.Run("all files from messages are collected", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		mproc := mock_processor.NewMockConversations(ctrl)
		dummyChannel := fixtures.DummyChannel("CTM1")
		mproc.EXPECT().
			ThreadMessages(gomock.Any(), "CTM1", testThread[0], false, true, testThread).
			Return(nil)

		mproc.EXPECT().
			Files(gomock.Any(), dummyChannel, testThread[1], testThread[1].Files).
			Return(nil)
		mproc.EXPECT().
			Files(gomock.Any(), dummyChannel, testThread[2], testThread[2].Files).
			Return(nil)

		if err := procThreadMsg(t.Context(), mproc, dummyChannel, testThread[0].ThreadTimestamp, false, true, testThread); err != nil {
			t.Fatal(err)
		}
	})
}

func Test_processLink(t *testing.T) {
	type args struct {
		item structures.EntityItem
	}
	tests := []struct {
		name              string
		args              args
		wantChanRequest   *request
		wantThreadRequest *request
		wantErr           bool
	}{
		{
			name: "channel",
			args: args{
				item: structures.EntityItem{Id: "CTM1"},
			},
			wantChanRequest: &request{
				sl: &structures.SlackLink{
					Channel: "CTM1",
				},
			},
			wantThreadRequest: nil,
			wantErr:           false,
		},
		{
			name: "channel URL",
			args: args{
				item: structures.EntityItem{Id: "https://test.slack.com/archives/CHYLGDP0D"},
			},
			wantChanRequest: &request{
				sl: &structures.SlackLink{
					Channel: "CHYLGDP0D",
				},
			},
			wantThreadRequest: nil,
			wantErr:           false,
		},
		{
			name: "thread URL",
			args: args{
				item: structures.EntityItem{Id: "https://test.slack.com/archives/CHYLGDP0D/p1610000000000000"},
			},
			wantChanRequest: nil,
			wantThreadRequest: &request{
				sl: &structures.SlackLink{
					Channel:  "CHYLGDP0D",
					ThreadTS: "1610000000.000000",
				},
				threadOnly: true,
			},
			wantErr: false,
		},
		{
			name: "thread Slackdump link URL",
			args: args{
				item: structures.EntityItem{Id: "CHYLGDP0D" + structures.LinkSep + "1577694990.000400"},
			},
			wantChanRequest: nil,
			wantThreadRequest: &request{
				sl: &structures.SlackLink{
					Channel:  "CHYLGDP0D",
					ThreadTS: "1577694990.000400",
				},
				threadOnly: true,
			},
			wantErr: false,
		},
		{
			"invalid link",
			args{
				item: structures.EntityItem{Id: "https://test.slack.com/archives/CHYLGDP0D/p1610000000000000/xxxx"},
			},
			nil,
			nil,
			true,
		},
		{
			name: "channel with oldest and latest set",
			args: args{
				item: structures.EntityItem{
					Id:     "CTM1",
					Oldest: time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC),
					Latest: time.Date(2021, 1, 8, 0, 0, 0, 0, time.UTC),
				},
			},
			wantChanRequest: &request{
				sl: &structures.SlackLink{
					Channel: "CTM1",
				},
				Oldest: time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC),
				Latest: time.Date(2021, 1, 8, 0, 0, 0, 0, time.UTC),
			},
			wantThreadRequest: nil,
			wantErr:           false,
		},
		{
			name: "thread with oldest and latest set",
			args: args{
				item: structures.EntityItem{
					Id:     "CTM1:1610000000.000000",
					Oldest: time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC),
					Latest: time.Date(2021, 1, 8, 0, 0, 0, 0, time.UTC),
				},
			},
			wantChanRequest: nil,
			wantThreadRequest: &request{
				sl: &structures.SlackLink{
					Channel:  "CTM1",
					ThreadTS: "1610000000.000000",
				},
				threadOnly: true,
				Oldest:     time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC),
				Latest:     time.Date(2021, 1, 8, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chans := make(chan request, 1)
			threads := make(chan request, 1)
			if err := processLink(chans, threads, tt.args.item); (err != nil) != tt.wantErr {
				t.Errorf("processLink() error = %v, wantErr %v", err, tt.wantErr)
				return // otherwise will block
			}
			if tt.wantErr {
				return // happy times
			}
			select {
			case got := <-chans:
				if !reflect.DeepEqual(&got, tt.wantChanRequest) {
					t.Errorf("processLink() got = %v, want %v", got, tt.wantChanRequest)
				}
			case got := <-threads:
				if !reflect.DeepEqual(&got, tt.wantThreadRequest) {
					t.Errorf("processLink() got = %v, want %v", got, tt.wantThreadRequest)
				}
			}
		})
	}
}

func TestStream_Users(t *testing.T) {
	ctx := t.Context()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		_, err := w.Write([]byte(`{"ok":false,"error":"not_authed"}`))
		if err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	l := rateLimits{
		users: network.NewLimiter(network.NoTier, 100, 100),
		tier:  network.DefLimits,
	}
	s := Stream{
		client: slack.New("test", slack.OptionAPIURL(srv.URL+"/")),
		limits: l,
	}
	m := mock_processor.NewMockUsers(gomock.NewController(t))
	err := s.Users(ctx, m)
	assert.Error(t, err)
}

func TestStream_ListChannels(t *testing.T) {
	testlimits := rateLimits{
		channels: network.NewLimiter(network.NoTier, 100, 100),
		tier:     network.DefLimits,
	}
	type args struct {
		ctx context.Context
		// proc processor.Channels
		p *slack.GetConversationsParameters
	}
	tests := []struct {
		name     string
		cs       *Stream
		args     args
		expectFn func(ms *mock_client.MockSlack, mc *mock_processor.MockChannels)
		wantErr  bool
	}{
		{
			name: "happy path",
			cs:   &Stream{limits: testlimits},
			args: args{ctx: t.Context(), p: &slack.GetConversationsParameters{}},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockChannels) {
				ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return(fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON), "", nil)
				mc.EXPECT().
					Channels(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "No channels returned, processor not called",
			cs:   &Stream{limits: testlimits},
			args: args{ctx: t.Context(), p: &slack.GetConversationsParameters{}},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockChannels) {
				ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return([]slack.Channel{}, "", nil)
			},
			wantErr: false,
		},
		{
			name: "next cursor causes another iteration",
			cs:   &Stream{limits: testlimits},
			args: args{ctx: t.Context(), p: &slack.GetConversationsParameters{}},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockChannels) {
				ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return(fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON), "next", nil)
				ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return([]slack.Channel{}, "", nil)
				mc.EXPECT().
					Channels(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "rate limiting error causes retry",
			cs:   &Stream{limits: testlimits},
			args: args{ctx: t.Context(), p: &slack.GetConversationsParameters{}},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockChannels) {
				call := ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return([]slack.Channel{}, "", &slack.RateLimitedError{RetryAfter: 100 * time.Millisecond})
				ms.EXPECT().
					GetConversationsContext(gomock.Any(), gomock.Any()).
					Return(fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON), "", nil).
					After(call)

				mc.EXPECT().
					Channels(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mc := mock_processor.NewMockChannels(ctrl)

			cs := tt.cs
			cs.client = ms
			if tt.expectFn != nil {
				tt.expectFn(ms, mc)
			}

			if err := cs.ListChannels(tt.args.ctx, mc, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Stream.ListChannels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStream_UsersBulk(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	testLimits := rateLimits{
		userinfo: network.NewLimiter(network.NoTier, 100, 100),
		tier:     network.DefLimits,
	}
	type fields struct {
		oldest time.Time
		latest time.Time
		// client     Slacker
		limits     rateLimits
		chanCache  *chanCache
		fastSearch bool
		inclusive  bool
		resultFn   []func(sr Result) error
	}
	type args struct {
		ctx context.Context
		// proc processor.Users
		ids []string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers)
		wantErr  bool
	}{
		{
			name:   "cancelled context",
			fields: fields{limits: testLimits},
			args: args{
				ctx: cancelled,
				ids: []string{"U12345678"},
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				mu.EXPECT().Users(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mu := mock_processor.NewMockUsers(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mu)
			}
			cs := &Stream{
				oldest:     tt.fields.oldest,
				latest:     tt.fields.latest,
				client:     ms,
				limits:     tt.fields.limits,
				chanCache:  tt.fields.chanCache,
				fastSearch: tt.fields.fastSearch,
				inclusive:  tt.fields.inclusive,
				resultFn:   tt.fields.resultFn,
			}
			if err := cs.UsersBulk(tt.args.ctx, mu, tt.args.ids...); (err != nil) != tt.wantErr {
				t.Errorf("Stream.UsersBulk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStream_UsersBulkWithCustom(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	testLimits := rateLimits{
		userinfo: network.NewLimiter(network.NoTier, 100, 100),
		tier:     network.DefLimits,
	}
	var (
		basicUser   = slack.User{ID: "U12345678", Profile: slack.UserProfile{FirstName: "Basic data"}}
		userProfile = slack.UserProfile{FirstName: "Full Data"}
	)
	type fields struct {
		oldest         time.Time
		latest         time.Time
		client         client.Slack
		limits         rateLimits
		chanCache      *chanCache
		fastSearch     bool
		inclusive      bool
		failChnlNotFnd bool
		resultFn       []func(sr Result) error
	}
	type args struct {
		ctx           context.Context
		proc          processor.Users
		includeLabels bool
		ids           []string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers)
		wantErr  bool
	}{
		{
			name:   "cancelled context",
			fields: fields{limits: testLimits},
			args: args{
				ctx: cancelled,
				ids: []string{"U12345678"},
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				mu.EXPECT().Users(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
		{
			name:   "user fetch ok, profile fetch ok - profile updated",
			fields: fields{limits: testLimits},
			args: args{
				ctx: context.Background(),
				ids: []string{"U12345678"},
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				ms.EXPECT().GetUserInfoContext(gomock.Any(), "U12345678").Return(&basicUser, nil)
				ms.EXPECT().GetUserProfileContext(gomock.Any(), &slack.GetUserProfileParameters{
					UserID:        "U12345678",
					IncludeLabels: false,
				}).Return(&userProfile, nil)

				wantUser := basicUser
				wantUser.Profile = userProfile // updated profile

				mu.EXPECT().Users(gomock.Any(), []slack.User{wantUser}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "propagates includeLabels",
			fields: fields{limits: testLimits},
			args: args{
				ctx:           context.Background(),
				ids:           []string{"U12345678"},
				includeLabels: true,
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				ms.EXPECT().GetUserInfoContext(gomock.Any(), "U12345678").Return(&basicUser, nil)
				ms.EXPECT().GetUserProfileContext(gomock.Any(), &slack.GetUserProfileParameters{
					UserID:        "U12345678",
					IncludeLabels: true,
				}).Return(&userProfile, nil)

				wantUser := basicUser
				wantUser.Profile = userProfile // updated profile

				mu.EXPECT().Users(gomock.Any(), []slack.User{wantUser}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "user fetch ok, profile fetch not ok - retains basic profile",
			fields: fields{limits: testLimits},
			args: args{
				ctx: context.Background(),
				ids: []string{"U12345678"},
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				ms.EXPECT().GetUserInfoContext(gomock.Any(), "U12345678").Return(&basicUser, nil)
				ms.EXPECT().GetUserProfileContext(gomock.Any(), &slack.GetUserProfileParameters{
					UserID:        "U12345678",
					IncludeLabels: false,
				}).Return(nil, errors.New("profile fetch error, no profile update expected"))

				wantUser := basicUser

				mu.EXPECT().Users(gomock.Any(), []slack.User{wantUser}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "user fetch not ok - error",
			fields: fields{limits: testLimits},
			args: args{
				ctx: context.Background(),
				ids: []string{"U12345678"},
			},
			expectFn: func(ms *mock_client.MockSlack, mu *mock_processor.MockUsers) {
				ms.EXPECT().GetUserInfoContext(gomock.Any(), "U12345678").Return(nil, errors.New("not your day"))
				ms.EXPECT().GetUserProfileContext(gomock.Any(), &slack.GetUserProfileParameters{
					UserID:        "U12345678",
					IncludeLabels: false,
				}).Return(&userProfile, nil)

				mu.EXPECT().Users(gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mu := mock_processor.NewMockUsers(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mu)
			}
			cs := &Stream{
				oldest:         tt.fields.oldest,
				latest:         tt.fields.latest,
				client:         ms,
				limits:         tt.fields.limits,
				chanCache:      tt.fields.chanCache,
				fastSearch:     tt.fields.fastSearch,
				inclusive:      tt.fields.inclusive,
				failChnlNotFnd: tt.fields.failChnlNotFnd,
				resultFn:       tt.fields.resultFn,
			}
			if err := cs.UsersBulkWithCustom(tt.args.ctx, mu, tt.args.includeLabels, tt.args.ids...); (err != nil) != tt.wantErr {
				t.Errorf("Stream.UsersBulkWithCustom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
