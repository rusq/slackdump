package bootstrap

import (
	"strings"
	"testing"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/client"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestSlackdumpSession(t *testing.T) {
	t.Run("no auth in context", func(t *testing.T) {
		_, err := SlackdumpSession(t.Context())
		if err == nil {
			t.Error("expected error")
		}
	})
	t.Run("auth in context", func(t *testing.T) {
		authJSON := `{"token":"` + strings.ReplaceAll(fixtures.TestClientToken, `xoxc`, `xoxb`) + `"}`
		prov, err := auth.Load(strings.NewReader(authJSON))
		if err != nil {
			t.Fatal(err)
		}

		// start fake Slack server
		srv := fixtures.TestAuthServer(t)
		defer srv.Close()
		s := client.Wrap(slack.New("", slack.OptionAPIURL(srv.URL+"/")))

		ctx := auth.WithContext(t.Context(), prov)
		if _, err := SlackdumpSession(ctx, slackdump.WithSlackClient(s)); err != nil {
			t.Error(err)
		}
	})
}
