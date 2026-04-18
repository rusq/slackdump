package edge

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
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

func TestNewWithClient_withTapeDoesNotCreateDefaultTape(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})

	tape := &closeTape{}
	cl, err := NewWithClient(
		"workspace",
		"T123",
		"xoxc-test",
		&http.Client{Transport: &closeTransport{}},
		WithTape(tape),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, cl.Close())
	})

	_, err = os.Stat(filepath.Join(tmp, "tape.txt"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}
