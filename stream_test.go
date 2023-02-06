package slackdump

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rusq/chttp"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/processors"
	"github.com/slack-go/slack"
)

func TestChannelStream(t *testing.T) {
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

	sd := slack.New(prov.SlackToken(), slack.OptionHTTPClient(chttp.Must(chttp.New("https://slack.com", prov.Cookies()))))

	cs := newChannelStream(sd, &DefOptions.Limits, time.Time{}, time.Time{})
	if err := cs.Stream(context.Background(), "D01MN4X7UGP", &processors.Discarder{}); err != nil {
		t.Fatal(err)
	}
}
