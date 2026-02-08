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
package fixtures

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/rusq/slack"
)

const (
	TestAppToken      = "xapp-1-A012RNBPFL3-1234567890123-c045facebeefbabecafef624ab2f2fe1cc640babf30e37e6b2d11c6094774782"
	TestBotToken      = "xoxb-123456789012-1234567890123-qCl4vKrWXWjArO5eoWgEUIPb"
	TestClientToken   = "xoxc-888888888888-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b"
	TestExportToken   = "xoxe-888888888888-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b"
	TestPersonalToken = "xoxp-777777777777-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b"
	TestOauthToken    = "xoxp-777777777777-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad8"
)

// InCI indicates whether the tests are running in CI.
var InCI = os.Getenv("CI") == "true"

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

// FilledFile returns a file that filled with sz bytes of 0x00.
func FilledFile(t *testing.T, sz int) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "sdunit*")
	if err != nil {
		panic(err)
	}
	if _, err := f.Write(bytes.Repeat([]byte{0x00}, sz)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	return f
}

// DummyChannel is the helper function that returns a pointer to a
// slack.Channel with the given ID, that could be used in tests.
func DummyChannel(id string) *slack.Channel {
	var ch slack.Channel
	ch.ID = id
	return &ch
}

// DebugTempDir creates a temporary directory for debugging purposes.
// It does not get removed after the test.
func DebugTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", t.Name()+"*")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tempdir: %s", d)
	return d
}

func SkipInCI(t *testing.T) {
	t.Helper()
	if InCI {
		t.Skip("skipping test in CI environment")
	}
}

func SkipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
}

func SkipIfNotExist(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("skipping, test file not found: %s", path)
		}
		t.Fatal(err)
	}
}

func SkipIfRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("skipping test as root")
	}
}
