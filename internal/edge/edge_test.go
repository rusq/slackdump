// Package edge provides a limited implementation of Slack edge api necessary
// to get the data from a slack workspace.
package edge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/auth"
)

var _ = godotenv.Load()

var (
	// preferrably guest workspace token.
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
	if testToken == "" {
		t.Skip("test token not set")
	}

	prov, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := New(context.Background(), prov)
	if err != nil {
		t.Fatal(err)
	}
	req := UsersListRequest{
		Channels: []string{"C6NL0QQSG"},
		Filter:   "everyone AND NOT bots AND NOT apps",
		Count:    20,
	}
	resp, err := cl.PostJSON(context.Background(), "/users/list", &req)
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
	if testToken == "" {
		t.Skip("test token not set")
	}
	au, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := New(context.Background(), au)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := cl.GetUsers(context.Background(), "U0LKLSNER", "U03K9GLS2", "U03KMNRQS")
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
