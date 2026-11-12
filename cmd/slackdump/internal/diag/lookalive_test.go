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

package diag

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

type fakePresenceAPI struct {
	up     *slack.UserPresence
	getErr error
	setErr error
	setTo  string
}

func (f *fakePresenceAPI) GetUserPresenceContext(context.Context, string) (*slack.UserPresence, error) {
	return f.up, f.getErr
}

func (f *fakePresenceAPI) SetUserPresenceContext(_ context.Context, p string) error {
	f.setTo = p
	return f.setErr
}

func Test_ensureAutoPresence(t *testing.T) {
	tests := []struct {
		name      string
		api       *fakePresenceAPI
		setAuto   bool
		wantSetTo string
		wantErr   bool
	}{
		{"auto presence, nothing to do", &fakePresenceAPI{up: &slack.UserPresence{Presence: "active"}}, true, "", false},
		{"manual away, no set-auto", &fakePresenceAPI{up: &slack.UserPresence{Presence: "away", ManualAway: true}}, false, "", false},
		{"manual away, set-auto", &fakePresenceAPI{up: &slack.UserPresence{Presence: "away", ManualAway: true}}, true, "auto", false},
		{"get error", &fakePresenceAPI{getErr: errors.New("boom")}, true, "", true},
		{"set error", &fakePresenceAPI{up: &slack.UserPresence{ManualAway: true}, setErr: errors.New("boom")}, true, "auto", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureAutoPresence(t.Context(), tt.api, "U1", tt.setAuto)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantSetTo, tt.api.setTo)
		})
	}
}

func Test_wsHeader(t *testing.T) {
	t.Run("with cookies", func(t *testing.T) {
		h := wsHeader([]*http.Cookie{{Name: "d", Value: "xoxd-abc%2F", Domain: ".slack.com"}, {Name: "d-s", Value: "123"}})
		assert.Equal(t, "d=xoxd-abc%2F; d-s=123", h.Get("Cookie"))
		assert.Equal(t, "https://app.slack.com", h.Get("Origin"))
		assert.NotEmpty(t, h.Get("User-Agent"))
	})
	t.Run("no cookies", func(t *testing.T) {
		h := wsHeader(nil)
		assert.Empty(t, h.Get("Cookie"))
	})
}
