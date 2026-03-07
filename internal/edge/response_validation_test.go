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

package edge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Methods_ValidateBaseResponse(t *testing.T) {
	type testCase struct {
		name     string
		endpoint string
		invoke   func(*Client) error
	}

	tests := []testCase{
		{
			name:     "ClientUserBoot",
			endpoint: "client.userBoot",
			invoke: func(cl *Client) error {
				_, err := cl.ClientUserBoot(t.Context())
				return err
			},
		},
		{
			name:     "IMList",
			endpoint: "im.list",
			invoke: func(cl *Client) error {
				_, err := cl.IMList(t.Context())
				return err
			},
		},
		{
			name:     "MPIMList",
			endpoint: "mpim.list",
			invoke: func(cl *Client) error {
				_, err := cl.MPIMList(t.Context())
				return err
			},
		},
		{
			name:     "ClientCounts",
			endpoint: "client.counts",
			invoke: func(cl *Client) error {
				_, err := cl.ClientCounts(t.Context())
				return err
			},
		},
		{
			name:     "ConversationsGenericInfo",
			endpoint: "conversations.genericInfo",
			invoke: func(cl *Client) error {
				_, err := cl.ConversationsGenericInfo(t.Context(), "C123")
				return err
			},
		},
		{
			name:     "ConversationsView",
			endpoint: "conversations.view",
			invoke: func(cl *Client) error {
				_, err := cl.ConversationsView(t.Context(), "C123")
				return err
			},
		},
		{
			name:     "ClientDMs",
			endpoint: "client.dms",
			invoke: func(cl *Client) error {
				_, err := cl.ClientDMs(t.Context())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				endpoint := strings.TrimPrefix(r.URL.Path, "/")
				if endpoint != tt.endpoint {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"ok":false,"error":"boom"}`))
			}))
			defer srv.Close()

			cl := &Client{
				cl:           http.DefaultClient,
				edgeAPI:      srv.URL + "/",
				webclientAPI: srv.URL + "/",
				token:        "xoxc-test",
			}

			err := tt.invoke(cl)
			if err == nil {
				t.Fatalf("%s() error = nil, want non-nil", tt.name)
			}
			if !strings.Contains(err.Error(), "boom") {
				t.Fatalf("%s() error = %q, want contains %q", tt.name, err.Error(), "boom")
			}
		})
	}
}
