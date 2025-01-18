package cache

import (
	"testing"

	"github.com/rusq/encio"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/types"
)

// testChannels is a test fixture for channels.
var testChannels = fixtures.Load[types.Channels](fixtures.TestChannels)

// TestSaveChannels tests that the saveChannels function works.
func TestSaveChannels(t *testing.T) {
	// test saving file works
	dir := t.TempDir()
	testfile := "test-chans.json"

	var m Manager
	assert.NoError(t, m.saveChannels(dir, testfile, testSuffix, testChannels))

	reopenedF, err := encio.Open(makeCacheFilename(dir, testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	cc, err := read[slack.Channel](reopenedF)
	assert.NoError(t, err)
	assert.Equal(t, testChannels, types.Channels(cc))
}
