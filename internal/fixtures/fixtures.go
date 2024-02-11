package fixtures

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/rusq/slack"
)

const (
	TestClientToken   = "xoxc-888888888888-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b"
	TestPersonalToken = "xoxp-777777777777-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b"
)

// Load loads a json data into T, or panics.
func Load[T any](js string) T {
	var ret T
	if err := json.Unmarshal([]byte(js), &ret); err != nil {
		panic(err)
	}
	return ret
}

// LoadPtr loads a json data into *T, or panics.
func LoadPtr[T any](js string) *T {
	v := Load[T](js)
	return &v
}

// FilledBuffer returns buffer that filled with sz bytes of 0x00.
func FilledBuffer(sz int) *bytes.Buffer {
	var buf bytes.Buffer
	buf.Write(bytes.Repeat([]byte{0x00}, sz))
	return &buf
}

func FilledFile(sz int) *os.File {
	f, err := os.CreateTemp("", "sdunit*")
	if err != nil {
		panic(err)
	}
	f.Write(bytes.Repeat([]byte{0x00}, sz))
	f.Seek(0, io.SeekStart)
	return f
}

// DummyChannel is the helper function that returns a pointer to a
// slack.Channel with the given ID, that could be used in tests.
func DummyChannel(id string) *slack.Channel {
	var ch slack.Channel
	ch.ID = id
	return &ch
}

func DebugTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", t.Name()+"*")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tempdir: %s", d)
	return d
}
