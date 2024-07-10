package slackdump

import (
	"context"
	"log"
	"math"
	"net/http"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/edge"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_auth"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_newLimiter(t *testing.T) {
	t.Parallel()
	type args struct {
		t     network.Tier
		burst uint
		boost int
	}
	tests := []struct {
		name      string
		args      args
		wantDelay time.Duration
	}{
		{
			"Tier test",
			args{
				network.Tier3,
				1,
				0,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3)*1000.0)) * time.Millisecond, // 6/5 sec
		},
		{
			"burst 2",
			args{
				network.Tier3,
				2,
				0,
			},
			1 * time.Millisecond,
		},
		{
			"boost 70",
			args{
				network.Tier3,
				1,
				70,
			},
			time.Duration(math.Round(60.0/float64(network.Tier3+70)*1000.0)) * time.Millisecond, // 500 msec
		},
	}
	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := network.NewLimiter(tt.args.t, tt.args.burst, tt.args.boost)

			assert.NoError(t, got.Wait(context.Background())) // prime

			start := time.Now()
			err := got.Wait(context.Background())
			stop := time.Now()

			assert.NoError(t, err)
			assert.WithinDurationf(t, start.Add(tt.wantDelay), stop, 10*time.Millisecond, "delayed for: %s, expected: %s", stop.Sub(start), tt.wantDelay)
		})
	}
}

func ExampleNew_tokenAndCookie() {
	provider, err := auth.NewValueAuth("xoxc-...", "xoxd-...")
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_cookieFile() {
	provider, err := auth.NewCookieFileAuth("xoxc-...", "cookies.txt")
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()

	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func ExampleNew_browserAuth() {
	provider, err := auth.NewBrowserAuth(context.Background())
	if err != nil {
		log.Print(err)
		return
	}
	fsa := openTempFS()
	defer fsa.Close()
	sd, err := New(context.Background(), provider)
	if err != nil {
		log.Print(err)
		return
	}
	_ = sd
}

func openTempFS() fsadapter.FSCloser {
	dir, err := os.MkdirTemp("", "slackdump")
	if err != nil {
		panic(err)
	}
	fsc, err := fsadapter.New(dir)
	if err != nil {
		panic(err)
	}
	return fsc
}

func TestSession_initWorkspaceInfo(t *testing.T) {
	ctx := context.Background()
	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mc := NewmockClienter(ctrl)
		mc.EXPECT().AuthTestContext(gomock.Any()).Return(&slack.AuthTestResponse{
			TeamID: "TEST",
		}, nil)
		s := Session{
			client: nil, // it should use the provided client
		}

		err := s.initWorkspaceInfo(ctx, mc)
		assert.NoError(t, err, "unexpected initialisation error")
	})
	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mc := NewmockClienter(ctrl)
		mc.EXPECT().AuthTestContext(gomock.Any()).Return(nil, assert.AnError)
		s := Session{
			client: nil, // it should use the provided client
		}
		err := s.initWorkspaceInfo(ctx, mc)
		assert.Error(t, err, "expected error")
	})
}

func TestSession_initClient(t *testing.T) {
	// fakeSlackAPI contains fake endpoints for the slack API.
	fakeSlackAPI := fstest.MapFS{
		"api/auth.test": &fstest.MapFile{
			Data: []byte(`{"ok":true,"url":"https:\/\/test.slack.com\/","team":"TEST","user":"test","team_id":"T123456","user_id":"U123456"}`),
			Mode: 0644,
		},
	}
	fakeEnterpriseSlackAPI := fstest.MapFS{
		"api/auth.test": &fstest.MapFile{
			Data: []byte(`{"ok":true,"url":"https:\/\/test.slack.com\/","team":"TEST","user":"test","team_id":"T123456","user_id":"U123456","enterprise_id":"E123456"}`),
		},
	}

	expectAuthTestFn := func(mc *mockClienter, enterpriseID string) {
		mc.EXPECT().AuthTestContext(gomock.Any()).Return(&slack.AuthTestResponse{
			TeamID:       "TEST",
			EnterpriseID: enterpriseID,
		}, nil)
	}
	t.Run("pre-initialised client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mc := NewmockClienter(ctrl)
		expectAuthTestFn(mc, "") // not an anterprise instance
		s := Session{
			client: mc,
		}
		err := s.initClient(context.Background(), nil, false)
		assert.NoError(t, err, "unexpected error")
		assert.IsType(t, &mockClienter{}, s.client)
	})
	t.Run("standard client", func(t *testing.T) {
		// http client will return the file from the fakeAPIFS.
		cl := http.Client{
			Transport: http.NewFileTransportFS(fakeSlackAPI),
		}

		ctrl := gomock.NewController(t)
		mprov := mock_auth.NewMockProvider(ctrl)
		mprov.EXPECT().SlackToken().Return("xoxb-...")
		mprov.EXPECT().HTTPClient().Return(&cl, nil)

		s := Session{
			client: nil,
			log:    logger.Default,
		}
		err := s.initClient(context.Background(), mprov, false)
		assert.NoError(t, err, "unexpected error")
		assert.IsType(t, &slack.Client{}, s.client)
	})

	t.Run("enterprise client", func(t *testing.T) {
		cl := http.Client{
			Transport: http.NewFileTransportFS(fakeEnterpriseSlackAPI),
		}

		ctrl := gomock.NewController(t)
		mprov := mock_auth.NewMockProvider(ctrl)
		mprov.EXPECT().SlackToken().Return("xoxb-...").Times(2)
		mprov.EXPECT().HTTPClient().Return(&cl, nil).Times(2)

		s := Session{
			client: nil,
			log:    logger.Default,
		}
		err := s.initClient(context.Background(), mprov, false)
		assert.NoError(t, err, "unexpected error")
		assert.IsType(t, &edge.Wrapper{}, s.client)
	})
	t.Run("forced enterprise client", func(t *testing.T) {
		cl := http.Client{
			Transport: http.NewFileTransportFS(fakeSlackAPI),
		}

		ctrl := gomock.NewController(t)
		mprov := mock_auth.NewMockProvider(ctrl)
		mprov.EXPECT().SlackToken().Return("xoxb-...").Times(2)
		mprov.EXPECT().HTTPClient().Return(&cl, nil).Times(2)

		s := Session{
			client: nil,
			log:    logger.Default,
		}
		err := s.initClient(context.Background(), mprov, true)
		assert.NoError(t, err, "unexpected error")
		assert.IsType(t, &edge.Wrapper{}, s.client)
	})
}
