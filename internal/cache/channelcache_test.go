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

package cache

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/types"
)

// testChannels is a test fixture for channels.
var testChannels = fixtures.Load[types.Channels](fixtures.TestChannelsJSON)

// TestSaveChannels tests that the saveChannels function works.
func TestSaveChannels(t *testing.T) {
	// test saving file works
	dir := t.TempDir()
	testfile := "test-chans.json"

	var m Manager
	assert.NoError(t, m.saveChannels(dir, testfile, testSuffix, testChannels))

	reopenedF, err := m.createOpener().Open(makeCacheFilename(dir, testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	cc, err := read[slack.Channel](reopenedF)
	assert.NoError(t, err)
	assert.Equal(t, testChannels, types.Channels(cc))
}
