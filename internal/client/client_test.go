package client

import (
	"errors"
	"net/http"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/edge"
)

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
