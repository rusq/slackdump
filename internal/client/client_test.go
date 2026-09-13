package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/edge"
	"github.com/rusq/slackdump/v4/internal/mocks/mock_auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeTransport struct {
	closed bool
	err    error
}

func (t *closeTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unexpected round trip")
}

func (t *closeTransport) Close() error {
	t.closed = true
	return t.err
}

type nopTape struct{}

func (nopTape) Write(p []byte) (int, error) {
	return len(p), nil
}

func (nopTape) Close() error {
	return nil
}

func TestClientClose_closesSlackTransportOnly(t *testing.T) {
	rt := &closeTransport{}
	cl := &Client{
		Client: slack.New("xoxc-test"),
		hcl:    &http.Client{Transport: rt},
	}

	err := cl.Close()

	require.NoError(t, err)
	assert.True(t, rt.closed)
}

func TestClientClose_closesSlackAndEdgeTransports(t *testing.T) {
	slackRT := &closeTransport{}
	edgeRT := &closeTransport{}
	ecl, err := edge.NewWithClient(
		"workspace",
		"T123",
		"xoxc-test",
		&http.Client{Transport: edgeRT},
		edge.WithTape(nopTape{}),
	)
	require.NoError(t, err)
	cl := &Client{
		Client: slack.New("xoxc-test"),
		hcl:    &http.Client{Transport: slackRT},
		edge:   ecl,
	}

	err = cl.Close()

	require.NoError(t, err)
	assert.True(t, slackRT.closed)
	assert.True(t, edgeRT.closed)
}

func TestClientClose_joinsErrors(t *testing.T) {
	slackErr := errors.New("slack close")
	edgeErr := errors.New("edge close")
	ecl, err := edge.NewWithClient(
		"workspace",
		"T123",
		"xoxc-test",
		&http.Client{Transport: &closeTransport{err: edgeErr}},
		edge.WithTape(nopTape{}),
	)
	require.NoError(t, err)
	cl := &Client{
		Client: slack.New("xoxc-test"),
		hcl:    &http.Client{Transport: &closeTransport{err: slackErr}},
		edge:   ecl,
	}

	err = cl.Close()

	require.Error(t, err)
	assert.ErrorIs(t, err, slackErr)
	assert.ErrorIs(t, err, edgeErr)
}

func TestClientClose_wrapIsNoOp(t *testing.T) {
	cl := Wrap(slack.New("xoxc-test"))

	assert.NoError(t, cl.Close())
}

func TestClient_CanvasThreadRoots(t *testing.T) {
	cl := &Client{Client: slack.New("xoxb-test")}

	assert.False(t, cl.CanvasSupported())
	_, err := cl.CanvasThreadRoots(t.Context(), "F123")
	require.ErrorIs(t, err, ErrOpNotSupported)
}

func TestNew(t *testing.T) {
	const clientToken = "xoxc-1-2-3-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name         string
		token        string
		enterpriseID string
		httpCalls    int
		wantCanvas   bool
		wantEdge     bool
	}{
		{
			name:       "standard workspace client token enables only canvas edge",
			token:      clientToken,
			httpCalls:  2,
			wantCanvas: true,
		},
		{
			name:      "bot token disables all canvas processing",
			token:     "xoxb-test",
			httpCalls: 1,
		},
		{
			name:         "enterprise client token shares edge client",
			token:        clientToken,
			enterpriseID: "E123",
			httpCalls:    2,
			wantCanvas:   true,
			wantEdge:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			prov := mock_auth.NewMockProvider(ctrl)
			prov.EXPECT().SlackToken().Return(tt.token).AnyTimes()
			authClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, "/api/auth.test", req.URL.Path)
				body := `{"ok":true,"url":"https://workspace.slack.com/","team_id":"T123","enterprise_id":"` + tt.enterpriseID + `"}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(body)),
					Request:    req,
				}, nil
			})}
			prov.EXPECT().HTTPClient().Return(authClient, nil).Times(tt.httpCalls)

			cl, err := New(t.Context(), prov)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCanvas, cl.CanvasSupported())
			assert.Equal(t, tt.wantEdge, cl.Edge() != nil)
			if tt.wantEdge {
				assert.Same(t, cl.edge, cl.canvas)
			}
		})
	}
}
