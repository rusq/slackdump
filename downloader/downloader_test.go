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

		dlqC := fltSeen(filesC)

		var got []Request
		for f := range dlqC {
			got = append(got, f)
		}
		assert.Equal(t, want, got)
	})
}

var benchInput = makeFileReqQ(100_000)

func BenchmarkFltSeen(b *testing.B) {

	inputC := make(chan Request)
	go func() {
		defer close(inputC)
		for _, req := range benchInput {
			inputC <- req
		}
	}()
	outputC := fltSeen(inputC)

	for n := 0; n < b.N; n++ {
		for out := range outputC {
			_ = out
		}
	}
}

func makeFileReqQ(numReq int) []Request {
	reqQ := make([]Request, numReq)
	for i := 0; i < numReq; i++ {
		reqQ[i] = Request{Fullpath: fixtures.RandString(8), URL: fixtures.RandString(16)}
	}
	return reqQ
}
