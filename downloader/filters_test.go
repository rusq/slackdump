package downloader

import (
	"testing"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func Test_fltSeen(t *testing.T) {
	t.Run("ensure that we don't get dup files", func(t *testing.T) {
		source := []fileRequest{
			{Directory: "x", File: &file1},
			{Directory: "x", File: &file2},
			{Directory: "a", File: &file2}, // this should appear, different dir
			{Directory: "x", File: &file3},
			{Directory: "x", File: &file3}, // duplicate
			{Directory: "x", File: &file3}, // duplicate
			{Directory: "x", File: &file4},
			{Directory: "x", File: &file5},
			{Directory: "y", File: &file5}, // this should appear, different dir
		}
		want := []fileRequest{
			{Directory: "x", File: &file1},
			{Directory: "x", File: &file2},
			{Directory: "a", File: &file2},
			{Directory: "x", File: &file3},
			{Directory: "x", File: &file4},
			{Directory: "x", File: &file5},
			{Directory: "y", File: &file5},
		}

		filesC := make(chan fileRequest)
		go func() {
			defer close(filesC)
			for _, f := range source {
				filesC <- f
			}
		}()

		c := Client{}
		dlqC := c.fltSeen(filesC)

		var got []fileRequest
		for f := range dlqC {
			got = append(got, f)
		}
		assert.Equal(t, want, got)
	})
}

func BenchmarkFltSeen(b *testing.B) {
	const numReq = 100_000
	input := makeFileReqQ(numReq, b.TempDir())
	inputC := make(chan fileRequest)
	go func() {
		defer close(inputC)
		for _, req := range input {
			inputC <- req
		}
	}()
	c := Client{}
	outputC := c.fltSeen(inputC)

	for n := 0; n < b.N; n++ {
		for out := range outputC {
			_ = out
		}
	}

}

func makeFileReqQ(numReq int, dir string) []fileRequest {
	reqQ := make([]fileRequest, numReq)
	for i := 0; i < numReq; i++ {
		reqQ[i] = randomFileReq(dir)
	}
	return reqQ
}

func randomFileReq(dirname string) fileRequest {
	return fileRequest{Directory: dirname, File: &slack.File{ID: fixtures.RandString(8), Name: fixtures.RandString(12)}}
}
