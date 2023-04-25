package slackdump

import (
	"context"
	"os"
	"path/filepath"
	"runtime/trace"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rusq/chttp"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/chunktest"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_processor"
)

var expandedLimits = Limits{
	Workers:         10,
	DownloadRetries: 10,
	Tier2: TierLimits{
		Boost:   100,
		Burst:   100,
		Retries: 20,
	},
	Tier3: TierLimits{
		Boost:   100,
		Burst:   100,
		Retries: 20,
	},
	Request: RequestLimit{
		Conversations: 200,
		Channels:      200,
		Replies:       1000,
	},
}

const testConversation = "C01SPFM1KNY"

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

	cs := newChannelStream(sd, &DefLimits)
	if err := cs.Conversations(context.Background(), rec, testConversation); err != nil {
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
	cs := newChannelStream(sd, &expandedLimits)
	if err := cs.Conversations(ctx, rec, fixtures.ChunkFileChannelID); err != nil {
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
		mproc.EXPECT().
			ThreadMessages(gomock.Any(), "CTM1", testThread[0], true, testThread[1:]).
			Return(nil)

		mproc.EXPECT().
			Files(gomock.Any(), "CTM1", testThread[1], true, testThread[1].Files).
			Return(nil)
		mproc.EXPECT().
			Files(gomock.Any(), "CTM1", testThread[2], true, testThread[2].Files).
			Return(nil)

		if err := procThreadMsg(context.Background(), mproc, fixtures.DummyChannel("CTM1"), testThread[0].ThreadTimestamp, true, testThread); err != nil {
			t.Fatal(err)
		}
	})
}
