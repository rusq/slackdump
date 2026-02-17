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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/structures"
)

func Test_pipelineFlags(t *testing.T) {
	tests := []struct {
		name      string
		chanTypes []string
		want      uint8
	}{
		{
			name:      "nil types means all default groups",
			chanTypes: nil,
			want:      runBoot | runIMs | runMPIMs | runAllConvs,
		},
		{
			name:      "empty types means all default groups",
			chanTypes: []string{},
			want:      runBoot | runIMs | runMPIMs | runAllConvs,
		},
		{
			name:      "im only",
			chanTypes: []string{structures.CIM},
			want:      runBoot | runIMs,
		},
		{
			name:      "mpim only",
			chanTypes: []string{structures.CMPIM},
			want:      runBoot | runMPIMs,
		},
		{
			name:      "public only",
			chanTypes: []string{structures.CPublic},
			want:      runBoot | runChannels,
		},
		{
			name:      "private only",
			chanTypes: []string{structures.CPrivate},
			want:      runBoot | runPrivate,
		},
		{
			name:      "public and private collapse to all conversations",
			chanTypes: []string{structures.CPublic, structures.CPrivate},
			want:      runBoot | runAllConvs,
		},
		{
			name:      "public private plus ims",
			chanTypes: []string{structures.CPrivate, structures.CPublic, structures.CIM, structures.CMPIM},
			want:      runBoot | runIMs | runMPIMs | runAllConvs,
		},
		{
			name:      "duplicates and order do not matter",
			chanTypes: []string{structures.CPublic, structures.CIM, structures.CPublic, structures.CIM},
			want:      runBoot | runIMs | runChannels,
		},
		{
			name:      "unknown types are ignored except boot remains",
			chanTypes: []string{"unknown", "other"},
			want:      runBoot,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineFlags(tt.chanTypes); got != tt.want {
				t.Errorf("pipelineFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pipelineFlags_DoesNotMutateInput(t *testing.T) {
	in := []string{structures.CPublic, structures.CIM, structures.CPublic}
	original := append([]string(nil), in...)

	_ = pipelineFlags(in)

	for i := range in {
		if in[i] != original[i] {
			t.Fatalf("input slice mutated at index %d: got %q, want %q", i, in[i], original[i])
		}
	}
}

func Test_plannedSteps(t *testing.T) {
	tests := []struct {
		name      string
		chanTypes []string
		want      []uint8
	}{
		{
			name:      "nil types use default all",
			chanTypes: nil,
			want:      []uint8{runBoot, runIMs, runMPIMs, runAllConvs},
		},
		{
			name:      "public and private collapse to all convs",
			chanTypes: []string{structures.CPrivate, structures.CPublic},
			want:      []uint8{runBoot, runAllConvs},
		},
		{
			name:      "public only with im",
			chanTypes: []string{structures.CPublic, structures.CIM},
			want:      []uint8{runBoot, runIMs, runChannels},
		},
		{
			name:      "private only with mpim",
			chanTypes: []string{structures.CPrivate, structures.CMPIM},
			want:      []uint8{runBoot, runMPIMs, runPrivate},
		},
		{
			name:      "unknown types still run boot only",
			chanTypes: []string{"unknown"},
			want:      []uint8{runBoot},
		},
		{
			name:      "order and duplicates do not affect planned step order",
			chanTypes: []string{structures.CIM, structures.CPrivate, structures.CIM},
			want:      []uint8{runBoot, runIMs, runPrivate},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plannedSteps(tt.chanTypes); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("plannedSteps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildPipeline_UsesPlannedSteps(t *testing.T) {
	tests := []struct {
		name      string
		chanTypes []string
		onlyMy    bool
	}{
		{name: "default", chanTypes: nil, onlyMy: false},
		{name: "public-only", chanTypes: []string{structures.CPublic}, onlyMy: true},
		{name: "private-only", chanTypes: []string{structures.CPrivate}, onlyMy: false},
		{name: "all-convs-explicit", chanTypes: []string{structures.CPublic, structures.CPrivate}, onlyMy: true},
		{name: "unknown", chanTypes: []string{"unknown"}, onlyMy: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &Client{}
			resultC := make(chan searchResult, maxPipelineSize)
			got := cl.buildPipeline(resultC, tt.chanTypes, tt.onlyMy)
			wantSteps := plannedSteps(tt.chanTypes)

			wantN := 0
			for _, step := range wantSteps {
				if step == runAllConvs {
					wantN += 2
					continue
				}
				wantN++
			}
			if len(got) != wantN {
				t.Fatalf("buildPipeline() len = %d, want %d", len(got), wantN)
			}
			for i, fn := range got {
				if fn == nil {
					t.Fatalf("buildPipeline() function at index %d is nil", i)
				}
			}
		})
	}
}

func TestClient_getConversationsContext_AllConvsUsesExplicitTypes(t *testing.T) {
	type state struct {
		mu          sync.Mutex
		channelType []string
	}
	st := &state{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimPrefix(r.URL.Path, "/")
		w.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))

		switch endpoint {
		case "client.userBoot":
			_, _ = io.WriteString(w, `{"ok":true,"channels":[]}`)
		case "im.list":
			_, _ = io.WriteString(w, `{"ok":true,"ims":[]}`)
		case "mpim.list":
			_, _ = io.WriteString(w, `{"ok":true,"groups":[]}`)
		case "search.modules.channels":
			ct := form.Get("channel_type")
			st.mu.Lock()
			st.channelType = append(st.channelType, ct)
			st.mu.Unlock()
			if ct == "" {
				_, _ = io.WriteString(w, `{"ok":false,"error":"no_channels_supplied"}`)
				return
			}
			_, _ = io.WriteString(w, `{"ok":true,"module":"channels","query":"","pagination":{"next_cursor":""},"items":[]}`)
		case "client.counts":
			_, _ = io.WriteString(w, `{"ok":true,"mpims":[]}`)
		case "conversations.genericInfo":
			_, _ = io.WriteString(w, `{"ok":true,"channels":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := &Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
		token:        "xoxc-test",
	}

	_, _, err := cl.getConversationsContext(t.Context(), &slack.GetConversationsParameters{}, true)
	if err != nil {
		t.Fatalf("getConversationsContext() error = %v", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	slices.Sort(st.channelType)
	want := []string{string(SCTPrivate), string(SCTPublic)}
	if !reflect.DeepEqual(st.channelType, want) {
		t.Fatalf("search channel_type calls = %v, want %v", st.channelType, want)
	}
}

func TestClient_getConversationsContext_DedupesAndFetchesMissingMPIM(t *testing.T) {
	type state struct {
		mu               sync.Mutex
		convsGenericJSON string
	}
	st := &state{
		convsGenericJSON: `{"ok":true,"channels":[{"id":"G2"}]}`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimPrefix(r.URL.Path, "/")
		w.Header().Set("Content-Type", "application/json")
		switch endpoint {
		case "client.userBoot":
			_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"CBOOT","name":"boot","is_channel":true},{"id":"X1","name":"dup","is_channel":true}]}`)
		case "im.list":
			_, _ = io.WriteString(w, `{"ok":true,"ims":[{"id":"D1","is_im":true},{"id":"X1","is_im":true}]}`)
		case "mpim.list":
			_, _ = io.WriteString(w, `{"ok":true,"groups":[{"id":"G1","name":"mpim","is_mpim":true}]}`)
		case "search.modules.channels":
			_, _ = io.WriteString(w, `{"ok":true,"module":"channels","query":"","pagination":{"next_cursor":""},"items":[{"id":"CSEARCH","is_channel":true},{"id":"X1","is_channel":true}]}`)
		case "client.counts":
			_, _ = io.WriteString(w, `{"ok":true,"mpims":[{"id":"G1"},{"id":"G2"}]}`)
		case "conversations.genericInfo":
			st.mu.Lock()
			resp := st.convsGenericJSON
			st.mu.Unlock()
			_, _ = io.WriteString(w, resp)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := &Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
		token:        "xoxc-test",
	}

	got, _, err := cl.getConversationsContext(t.Context(), &slack.GetConversationsParameters{}, false)
	if err != nil {
		t.Fatalf("getConversationsContext() error = %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("getConversationsContext() channel count = %d, want 6", len(got))
	}

	ids := make([]string, 0, len(got))
	for _, ch := range got {
		ids = append(ids, ch.ID)
	}
	slices.Sort(ids)
	wantIDs := []string{"CBOOT", "CSEARCH", "D1", "G1", "G2", "X1"}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("getConversationsContext() IDs = %v, want %v", ids, wantIDs)
	}
}

func TestClient_GetConversationsContextEx_OnlyMyPassedToSearch(t *testing.T) {
	type state struct {
		mu             sync.Mutex
		searchOnlyMy   []string
		searchType     []string
		genericReqJSON string
	}
	st := &state{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimPrefix(r.URL.Path, "/")
		w.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))

		switch endpoint {
		case "client.userBoot":
			_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"CBOOT","name":"boot","is_channel":true}]}`)
		case "search.modules.channels":
			st.mu.Lock()
			st.searchOnlyMy = append(st.searchOnlyMy, form.Get("search_only_my_channels"))
			st.searchType = append(st.searchType, form.Get("channel_type"))
			st.mu.Unlock()
			_, _ = io.WriteString(w, `{"ok":true,"module":"channels","query":"","pagination":{"next_cursor":""},"items":[{"id":"C1","is_channel":true}]}`)
		case "client.counts":
			_, _ = io.WriteString(w, `{"ok":true,"mpims":[]}`)
		case "conversations.genericInfo":
			st.mu.Lock()
			st.genericReqJSON = form.Get("updated_channels")
			st.mu.Unlock()
			_, _ = io.WriteString(w, `{"ok":true,"channels":[]}`)
		default:
			_, _ = io.WriteString(w, `{"ok":true}`)
		}
	}))
	defer srv.Close()

	cl := &Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
		token:        "xoxc-test",
	}

	_, _, err := cl.GetConversationsContextEx(
		t.Context(),
		&slack.GetConversationsParameters{Types: []string{structures.CPublic}},
		true,
	)
	if err != nil {
		t.Fatalf("GetConversationsContextEx() error = %v", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.searchOnlyMy) != 1 {
		t.Fatalf("search.modules.channels calls = %d, want 1", len(st.searchOnlyMy))
	}
	if st.searchOnlyMy[0] != "true" {
		t.Fatalf("search_only_my_channels = %q, want %q", st.searchOnlyMy[0], "true")
	}
	if st.searchType[0] != string(SCTPublic) {
		t.Fatalf("channel_type = %q, want %q", st.searchType[0], string(SCTPublic))
	}
	if st.genericReqJSON != "{}" {
		t.Fatalf("updated_channels = %q, want {}", st.genericReqJSON)
	}
}

func TestClient_getConversationsContext_ErrorFromPipelineAndPostProcessing(t *testing.T) {
	tests := []struct {
		name        string
		handler     func(http.ResponseWriter, *http.Request)
		wantErrPart string
	}{
		{
			name: "pipeline error from search parse",
			handler: func(w http.ResponseWriter, r *http.Request) {
				endpoint := strings.TrimPrefix(r.URL.Path, "/")
				w.Header().Set("Content-Type", "application/json")
				switch endpoint {
				case "client.userBoot":
					_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"CBOOT","name":"boot","is_channel":true}]}`)
				case "im.list", "mpim.list":
					_, _ = io.WriteString(w, `{"ok":true}`)
				case "search.modules.channels":
					_, _ = io.WriteString(w, `{"ok":true`) // invalid json
				default:
					_, _ = io.WriteString(w, `{"ok":true,"channels":[],"mpims":[]}`)
				}
			},
			wantErrPart: "unexpected EOF",
		},
		{
			name: "post-processing error from client.counts",
			handler: func(w http.ResponseWriter, r *http.Request) {
				endpoint := strings.TrimPrefix(r.URL.Path, "/")
				w.Header().Set("Content-Type", "application/json")
				switch endpoint {
				case "client.userBoot":
					_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"CBOOT","name":"boot","is_channel":true}]}`)
				case "im.list", "mpim.list":
					_, _ = io.WriteString(w, `{"ok":true}`)
				case "search.modules.channels":
					_, _ = io.WriteString(w, `{"ok":true,"module":"channels","query":"","pagination":{"next_cursor":""},"items":[]}`)
				case "client.counts":
					_, _ = io.WriteString(w, `{"ok":false,"error":"counts_failed"}`)
				default:
					_, _ = io.WriteString(w, `{"ok":true,"channels":[]}`)
				}
			},
			wantErrPart: "counts_failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer srv.Close()

			cl := &Client{
				cl:           http.DefaultClient,
				edgeAPI:      srv.URL + "/",
				webclientAPI: srv.URL + "/",
				token:        "xoxc-test",
			}

			_, _, err := cl.getConversationsContext(context.Background(), &slack.GetConversationsParameters{}, false)
			if err == nil {
				t.Fatalf("getConversationsContext() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErrPart)
			}
		})
	}
}

func TestClient_getConversationsContext_FetchesOnlyMissingMPIMIDs(t *testing.T) {
	type state struct {
		mu       sync.Mutex
		fetchIDs []string
	}
	st := &state{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.TrimPrefix(r.URL.Path, "/")
		w.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))

		switch endpoint {
		case "client.userBoot":
			_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"G_SEEN","name":"seen","is_mpim":true}]}`)
		case "im.list", "mpim.list":
			_, _ = io.WriteString(w, `{"ok":true}`)
		case "search.modules.channels":
			_, _ = io.WriteString(w, `{"ok":true,"module":"channels","query":"","pagination":{"next_cursor":""},"items":[]}`)
		case "client.counts":
			_, _ = io.WriteString(w, `{"ok":true,"mpims":[{"id":"G_SEEN"},{"id":"G_MISSING"}]}`)
		case "conversations.genericInfo":
			raw := form.Get("updated_channels")
			m := map[string]int{}
			_ = json.Unmarshal([]byte(raw), &m)
			ids := make([]string, 0, len(m))
			for id := range m {
				ids = append(ids, id)
			}
			slices.Sort(ids)
			st.mu.Lock()
			st.fetchIDs = ids
			st.mu.Unlock()
			_, _ = io.WriteString(w, `{"ok":true,"channels":[{"id":"G_MISSING"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cl := &Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
		token:        "xoxc-test",
	}

	_, _, err := cl.getConversationsContext(t.Context(), &slack.GetConversationsParameters{}, false)
	if err != nil {
		t.Fatalf("getConversationsContext() error = %v", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	want := []string{"G_MISSING"}
	if !reflect.DeepEqual(st.fetchIDs, want) {
		t.Fatalf("conversations.genericInfo IDs = %v, want %v", st.fetchIDs, want)
	}
}
