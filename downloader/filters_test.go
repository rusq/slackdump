package downloader

import (
	"testing"

	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/stretchr/testify/assert"
)

func Test_fltSeen(t *testing.T) {
	t.Run("ensure that we don't get dup files", func(t *testing.T) {
		source := []Request{
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"}, // different path
			{Fullpath: "x/file3", URL: "url3"},
			{Fullpath: "x/file4", URL: "url4"},
			{Fullpath: "x/file5", URL: "url5"},
			{Fullpath: "y/file5", URL: "url5"},
			{Fullpath: "x/file1", URL: "url2"}, // different url same path
			// duplicates
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"},
		}
		want := []Request{
			{Fullpath: "x/file1", URL: "url1"},
			{Fullpath: "x/file2", URL: "url2"},
			{Fullpath: "a/file2", URL: "url2"},
			{Fullpath: "x/file3", URL: "url3"},
			{Fullpath: "x/file4", URL: "url4"},
			{Fullpath: "x/file5", URL: "url5"},
			{Fullpath: "y/file5", URL: "url5"},
			{Fullpath: "x/file1", URL: "url2"},
		}

		filesC := make(chan Request)
		go func() {
			defer close(filesC)
			for _, f := range source {
				filesC <- f
			}
		}()

		c := Client{}
		dlqC := c.fltSeen(filesC)

		var got []Request
		for f := range dlqC {
			got = append(got, f)
		}
		assert.Equal(t, want, got)
	})
}

func BenchmarkFltSeen(b *testing.B) {
	const numReq = 100_000
	input := makeFileReqQ(numReq, b.TempDir())
	inputC := make(chan Request)
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

func makeFileReqQ(numReq int, dir string) []Request {
	reqQ := make([]Request, numReq)
	for i := 0; i < numReq; i++ {
		reqQ[i] = randomFileReq(dir)
	}
	return reqQ
}

func randomFileReq(dirname string) Request {
	return Request{Fullpath: fixtures.RandString(8), URL: fixtures.RandString(16)}
}
