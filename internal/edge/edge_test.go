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

// Package edge provides a limited implementation of Slack edge api necessary
// to get the data from a slack workspace.
package edge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/fixtures"
)

var _ = godotenv.Load()

var (
	// preferably guest workspace token.
	testToken  = os.Getenv("EDGE_TOKEN")
	testCookie = os.Getenv("EDGE_COOKIE")
)

func testServer(status int, payload []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write(payload)
	}))
}

func TestNew(t *testing.T) {
	fixtures.SkipInCI(t)
	if testToken == "" {
		t.Skip("test token not set")
	}

	prov, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := New(t.Context(), prov)
	if err != nil {
		t.Fatal(err)
	}
	req := UsersListRequest{
		Channels: []string{"C6NL0QQSG"},
		Filter:   "everyone AND NOT bots AND NOT apps",
		Count:    20,
	}
	resp, err := cl.PostJSON(t.Context(), "/users/list", &req)
	if err != nil {
		t.Fatal(err)
	}
	var ur UsersListResponse
	if err := cl.ParseResponse(&ur, resp); err != nil {
		t.Fatal(err)
	}
	t.Error(ur)
}

func TestGetUsers(t *testing.T) {
	fixtures.SkipInCI(t)
	if testToken == "" {
		t.Skip("test token not set")
	}
	au, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := New(t.Context(), au)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := cl.GetUsers(t.Context(), "U0LKLSNER", "U03K9GLS2", "U03KMNRQS")
	if err != nil {
		t.Fatal(err)
	}
	t.Error(ui)
}

func Test_values(t *testing.T) {
	type args struct {
		s         any
		omitempty bool
	}
	tests := []struct {
		name string
		args args
		want url.Values
	}{
		{
			name: "empty",
			args: args{
				s:         conversationsGenericInfoForm{},
				omitempty: true,
			},
			want: url.Values{
				"_x_app_name":      []string{""},
				"_x_sonic":         []string{"false"},
				"token":            []string{""},
				"updated_channels": []string{""},
			},
		},
		{
			name: "converts fields to url.Values",
			args: args{
				s: conversationsGenericInfoForm{
					BaseRequest: BaseRequest{
						Token: "token-value",
					},
					UpdatedChannels: `{"C0412851":0}`,
					WebClientFields: webclientReason("fallback:UnknownFetchManager"),
				},
				omitempty: true,
			},
			want: url.Values{
				"token":            []string{"token-value"},
				"updated_channels": []string{`{"C0412851":0}`},
				"_x_reason":        []string{"fallback:UnknownFetchManager"},
				"_x_mode":          []string{"online"},
				"_x_sonic":         []string{"true"},
				"_x_app_name":      []string{"client"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := values(tt.args.s, tt.args.omitempty)
			assert.Equal(t, tt.want, got)
		})
	}
}

// replayServer responds with statuses in order (the last one repeats) and
// records every request body and content type it receives.
type replayServer struct {
	*httptest.Server
	mu       sync.Mutex
	bodies   []string
	ctypes   []string
	statuses []int
	// retryAfter is the Retry-After header value per 429 response (the last
	// one repeats), "-" omits the header.  Defaults to "0".
	retryAfter []string
}

func newReplayServer(t *testing.T, statuses ...int) *replayServer {
	t.Helper()
	rs := &replayServer{statuses: statuses}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rs.mu.Lock()
		n := len(rs.bodies)
		rs.bodies = append(rs.bodies, string(b))
		rs.ctypes = append(rs.ctypes, r.Header.Get(hdrContentType))
		rs.mu.Unlock()
		status := rs.statuses[min(n, len(rs.statuses)-1)]
		if status == http.StatusTooManyRequests {
			ra := "0"
			if len(rs.retryAfter) > 0 {
				ra = rs.retryAfter[min(n, len(rs.retryAfter)-1)]
			}
			if ra != "-" {
				w.Header().Set("Retry-After", ra)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(rs.Close)
	return rs
}

func (rs *replayServer) requests() ([]string, []string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]string(nil), rs.bodies...), append([]string(nil), rs.ctypes...)
}

func testClient(tape io.WriteCloser) *Client {
	cl := &Client{cl: http.DefaultClient, token: "xoxc-test", tape: nopTape{}}
	if tape != nil {
		cl.tape = tape
	}
	return cl
}

func TestClient_do(t *testing.T) {
	const body = "token=xoxc-test&channel=C123"
	t.Run("rate limited, then ok: retry resends the same body", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests, http.StatusOK)
		resp, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		require.NoError(t, err)
		resp.Body.Close()
		bodies, _ := srv.requests()
		assert.Equal(t, []string{body, body}, bodies)
	})
	t.Run("rate limited twice: returns RateLimitedError", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests)
		_, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		_, ok := err.(*slack.RateLimitedError)
		assert.True(t, ok, "got %T: %v", err, err)
		bodies, _ := srv.requests()
		assert.Equal(t, []string{body, body}, bodies)
	})
	t.Run("rate limited twice: returns Retry-After without waiting it", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests)
		srv.retryAfter = []string{"0", "30"}
		start := time.Now()
		_, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		var rle *slack.RateLimitedError
		require.True(t, errors.As(err, &rle), "got %v", err)
		assert.Equal(t, 30*time.Second, rle.RetryAfter)
		assert.Less(t, time.Since(start), time.Second)
	})
	t.Run("rate limited twice without Retry-After: returns default delay", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests)
		srv.retryAfter = []string{"0", "-"}
		_, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		var rle *slack.RateLimitedError
		require.True(t, errors.As(err, &rle), "got %v", err)
		assert.Equal(t, defRetryAfter, rle.RetryAfter)
	})
	t.Run("server error: no retry, bare StatusCodeError", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusServiceUnavailable)
		_, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		sce, ok := err.(slack.StatusCodeError)
		require.True(t, ok, "got %T: %v", err, err)
		assert.Equal(t, http.StatusServiceUnavailable, sce.Code)
		bodies, _ := srv.requests()
		assert.Len(t, bodies, 1)
	})
	t.Run("cancelled while waiting for Retry-After", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests)
		srv.retryAfter = []string{"30"}
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		_, err := testClient(nil).do(ctx, srv.URL, "application/x-www-form-urlencoded", []byte(body))
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		bodies, _ := srv.requests()
		assert.Len(t, bodies, 1)
	})
	t.Run("non-recoverable status: no retry", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusBadRequest)
		_, err := testClient(nil).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		var sce slack.StatusCodeError
		require.True(t, errors.As(err, &sce), "got %v", err)
		assert.Equal(t, http.StatusBadRequest, sce.Code)
		bodies, _ := srv.requests()
		assert.Len(t, bodies, 1)
	})
	t.Run("cancelled context", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusOK)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := testClient(nil).do(ctx, srv.URL, "application/x-www-form-urlencoded", []byte(body))
		assert.ErrorIs(t, err, context.Canceled)
	})
	t.Run("tape records the body once across retries", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusTooManyRequests, http.StatusOK)
		tape := new(closeTape)
		resp, err := testClient(tape).do(t.Context(), srv.URL, "application/x-www-form-urlencoded", []byte(body))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, body+"\n\n", tape.String())
	})
	t.Run("tape records nothing if the request was not sent", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusOK)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		tape := new(closeTape)
		_, err := testClient(tape).do(ctx, srv.URL, "application/x-www-form-urlencoded", []byte(body))
		require.Error(t, err)
		assert.Empty(t, tape.String())
	})
}

func TestClient_PostFormRaw(t *testing.T) {
	t.Run("sets token and content type", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusOK)
		resp, err := testClient(nil).PostFormRaw(t.Context(), srv.URL, url.Values{"channel": {"C123"}})
		require.NoError(t, err)
		resp.Body.Close()
		bodies, ctypes := srv.requests()
		want := url.Values{"channel": {"C123"}, "token": {"xoxc-test"}}.Encode()
		assert.Equal(t, []string{want}, bodies)
		assert.Equal(t, []string{"application/x-www-form-urlencoded"}, ctypes)
	})
}

func TestClient_PostJSON(t *testing.T) {
	t.Run("sets token and content type", func(t *testing.T) {
		srv := newReplayServer(t, http.StatusOK)
		cl := testClient(nil)
		cl.edgeAPI = srv.URL + "/"
		resp, err := cl.PostJSON(t.Context(), "users/info", &BaseRequest{})
		require.NoError(t, err)
		resp.Body.Close()
		bodies, ctypes := srv.requests()
		assert.Equal(t, []string{`{"token":"xoxc-test"}`}, bodies)
		assert.Equal(t, []string{"application/json"}, ctypes)
	})
}

func TestClient_callEdgeAPI(t *testing.T) {
	t.Run("connection closed without response: returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		}))
		t.Cleanup(srv.Close)
		cl := testClient(nil)
		cl.edgeAPI = srv.URL + "/"
		var v map[string]any
		assert.NotPanics(t, func() {
			err := cl.callEdgeAPI(t.Context(), &v, "users/info", &BaseRequest{})
			assert.Error(t, err)
		})
	})
}

func Test_parseRetryAfter(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", header: "30", want: 30 * time.Second},
		{name: "zero", header: "0", want: 0},
		{name: "missing", header: "", wantErr: true},
		{name: "negative", header: "-5", wantErr: true},
		{name: "garbage", header: "soon", wantErr: true},
		{name: "fractional seconds", header: "1.5", want: 1500 * time.Millisecond},
		{name: "max", header: "3600", want: maxRetryAfter},
		{name: "above max", header: "3601", wantErr: true},
		{name: "overflows duration", header: "9300000000", wantErr: true},
		{name: "NaN", header: "NaN", wantErr: true},
		{name: "date too far in the future", header: "Fri, 01 Jan 2100 00:00:00 GMT", wantErr: true},
		{name: "date in the past", header: "Wed, 21 Oct 2015 07:28:00 GMT", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}
			got, err := parseRetryAfter(resp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	t.Run("date in the future", func(t *testing.T) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", time.Now().Add(time.Minute).UTC().Format(http.TimeFormat))
		got, err := parseRetryAfter(resp)
		require.NoError(t, err)
		assert.InDelta(t, float64(time.Minute), float64(got), float64(2*time.Second))
	})
}
