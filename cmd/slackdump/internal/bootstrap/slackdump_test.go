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
package bootstrap

import (
	"strings"
	"testing"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/fixtures"
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
