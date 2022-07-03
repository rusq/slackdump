package slackdump

import (
	"path"
	"sync"
	"testing"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/types"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestSession_pipeFiles(t *testing.T) {
	var (
		file1 = slack.File{ID: "f1", Name: "filename1.ext", URLPrivateDownload: "https://file1_url", Size: 100}
		file2 = slack.File{ID: "f2", Name: "filename2.ext", URLPrivateDownload: "https://file2_url", Size: 200}
		file3 = slack.File{ID: "f3", Name: "filename3.ext", URLPrivateDownload: "https://file3_url", Size: 300}
		file4 = slack.File{ID: "f4", Name: "filename4.ext", URLPrivateDownload: "https://file4_url", Size: 400}
		file5 = slack.File{ID: "f5", Name: "filename5.ext", URLPrivateDownload: "https://file5_url", Size: 500}
		file6 = slack.File{ID: "f6", Name: "filename6.ext", URLPrivateDownload: "https://file6_url", Size: 600}
	)

	var (
		testFileMsg1 = types.Message{
			Message: slack.Message{
				Msg: slack.Msg{
					ClientMsgID: "1",
					Channel:     "x",
					Type:        "y",
					Files: []slack.File{
						file1, file2, file3,
					}},
			}}
		testFileMsg2 = types.Message{
			Message: slack.Message{
				Msg: slack.Msg{
					ClientMsgID: "2",
					Channel:     "x",
					Type:        "z",
					Files: []slack.File{
						file4, file5, file6,
					}},
			}}
	)

	t.Run("ensure all files make it to channel", func(t *testing.T) {
		want := []slack.File{
			file1, file2, file3, file4, file5, file6,
		}
		msgs := []types.Message{testFileMsg1, testFileMsg2}
		got := pipeTestSuite(t, msgs, "test")

		assert.Equal(t, want, got)
	})
	t.Run("ensure message URLs are updated", func(t *testing.T) {
		const testDir = "test_dir"
		msgs := []types.Message{testFileMsg1, testFileMsg2}

		wantURLs := []string{
			path.Join(testDir, downloader.Filename(&file1)),
			path.Join(testDir, downloader.Filename(&file2)),
			path.Join(testDir, downloader.Filename(&file3)),
			path.Join(testDir, downloader.Filename(&file4)),
			path.Join(testDir, downloader.Filename(&file5)),
			path.Join(testDir, downloader.Filename(&file6)),
		}
		_ = pipeTestSuite(t, msgs, testDir)
		idx := 0
		for _, m := range msgs {
			for _, f := range m.Files {
				wantURL := wantURLs[idx]
				assert.Equal(t, wantURL, f.URLPrivateDownload, "private Download URL mismatch")
				assert.Equal(t, wantURL, f.URLPrivate, "private URL mismatch")
				idx++
			}
		}
	})
}

func pipeTestSuite(t *testing.T, msgs []types.Message, dir string) []slack.File {
	var wg sync.WaitGroup

	var got []slack.File
	filesC := make(chan *slack.File)
	go func(c <-chan *slack.File) {
		// catcher
		for f := range c {
			got = append(got, *f)
		}
		wg.Done()
	}(filesC)
	wg.Add(1)

	pipeAndUpdateFiles(filesC, msgs, dir)
	close(filesC)
	wg.Wait()
	return got
}
