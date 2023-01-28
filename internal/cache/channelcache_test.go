package cache

import (
	"testing"

	"github.com/rusq/encio"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

// testChannels is a test fixture for channels.
var testChannels = fixtures.Load[types.Channels](fixtures.TestChannels)

// TestSaveChannels tests that the saveChannels function works.
func TestSaveChannels(t *testing.T) {
	// test saving file works
	dir := t.TempDir()
	testfile := "test-chans.json"

	assert.NoError(t, saveChannels(dir, testfile, testSuffix, testChannels))

	reopenedF, err := encio.Open(makeCacheFilename(dir, testfile, testSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer reopenedF.Close()
	cc, err := read[slack.Channel](reopenedF)
	assert.NoError(t, err)
	assert.Equal(t, testChannels, types.Channels(cc))
}
