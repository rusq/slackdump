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

package slackdump

import (
	"path"
	"sync"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/downloader"
	"github.com/rusq/slackdump/v4/types"
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
		want := []downloader.Request{
			{Fullpath: "test/f1-filename1.ext", URL: file1.URLPrivateDownload},
			{Fullpath: "test/f2-filename2.ext", URL: file2.URLPrivateDownload},
			{Fullpath: "test/f3-filename3.ext", URL: file3.URLPrivateDownload},
			{Fullpath: "test/f4-filename4.ext", URL: file4.URLPrivateDownload},
			{Fullpath: "test/f5-filename5.ext", URL: file5.URLPrivateDownload},
			{Fullpath: "test/f6-filename6.ext", URL: file6.URLPrivateDownload},
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

func pipeTestSuite(t *testing.T, msgs []types.Message, dir string) []downloader.Request {
	t.Helper()
	var wg sync.WaitGroup

	var got []downloader.Request
	filesC := make(chan downloader.Request)
	go func(c <-chan downloader.Request) {
		// catcher
		for f := range c {
			got = append(got, f)
		}
		wg.Done()
	}(filesC)
	wg.Add(1)

	pipeAndUpdateFiles(filesC, msgs, dir)
	close(filesC)
	wg.Wait()
	return got
}
