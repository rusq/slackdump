package edge

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type closeTape struct {
	closed bool
	err    error
}

func (t *closeTape) Write(p []byte) (int, error) {
	return len(p), nil
}

func (t *closeTape) Close() error {
	t.closed = true
	return t.err
}

func TestClientClose_closesTapeAndTransport(t *testing.T) {
	tape := &closeTape{}
	rt := &closeTransport{}
	cl := &Client{
		tape: tape,
		cl:   &http.Client{Transport: rt},
	}

	err := cl.Close()

	require.NoError(t, err)
	assert.True(t, tape.closed)
	assert.True(t, rt.closed)
}

func TestClientClose_joinsErrors(t *testing.T) {
	tapeErr := errors.New("tape close")
	transportErr := errors.New("transport close")
	cl := &Client{
		tape: &closeTape{err: tapeErr},
		cl:   &http.Client{Transport: &closeTransport{err: transportErr}},
	}

	err := cl.Close()

	require.Error(t, err)
	assert.ErrorIs(t, err, tapeErr)
	assert.ErrorIs(t, err, transportErr)
}
