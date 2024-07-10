package stream

import (
	"context"
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
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	prov, err := m.Auth(context.Background(), wsp, nil)
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

	cs := New(sd, &network.DefLimits)
	if err := cs.SyncConversations(context.Background(), rec, testConversation); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderStream(t *testing.T) {
	ctx, task := trace.NewTask(context.Background(), "TestRecorderStream")
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
	cs := New(sd, &network.NoLimits)
	if err := cs.SyncConversations(ctx, rec, fixtures.ChunkFileChannelID); err != nil {
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
			ThreadMessages(gomock.Any(), "CTM1", testThread[0], false, true, testThread[1:]).
			Return(nil)

		mproc.EXPECT().
			Files(gomock.Any(), dummyChannel, testThread[1], testThread[1].Files).
			Return(nil)
		mproc.EXPECT().
			Files(gomock.Any(), dummyChannel, testThread[2], testThread[2].Files).
			Return(nil)

		if err := procThreadMsg(context.Background(), mproc, dummyChannel, testThread[0].ThreadTimestamp, false, true, testThread); err != nil {
			t.Fatal(err)
		}
	})
}

func Test_processLink(t *testing.T) {
	type args struct {
		link string
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
				link: "CTM1",
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
				link: "https://test.slack.com/archives/CHYLGDP0D",
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
				link: "https://test.slack.com/archives/CHYLGDP0D/p1610000000000000",
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
				link: "CHYLGDP0D" + structures.LinkSep + "1577694990.000400",
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
				link: "https://test.slack.com/archives/CHYLGDP0D/p1610000000000000/xxxx",
			},
			nil,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chans := make(chan request, 1)
			threads := make(chan request, 1)
			if err := processLink(chans, threads, tt.args.link); (err != nil) != tt.wantErr {
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
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":false,"error":"not_authed"}`))
	}))
	defer srv.Close()
	l := rateLimits{
		users: network.NewLimiter(network.NoTier, 100, 100),
		tier:  &network.DefLimits,
	}
	s := Stream{
		client: slack.New("test", slack.OptionAPIURL(srv.URL+"/")),
		limits: l,
	}
	m := mock_processor.NewMockUsers(gomock.NewController(t))
	err := s.Users(ctx, m)
	assert.Error(t, err)
}
