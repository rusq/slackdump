package slackdump

import (
	"context"
	"runtime/trace"

	"github.com/rusq/slackdump/downloader"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

// SaveFileTo saves a single file to the specified directory.
func (sd *SlackDumper) SaveFileTo(ctx context.Context, dir string, f *slack.File) (int64, error) {
	dl := downloader.New(
		sd.client,
		downloader.Limiter(newLimiter(noTier, sd.options.Tier3Burst, 0)),
		downloader.Retries(sd.options.DownloadRetries),
		downloader.Workers(sd.options.Workers),
	)
	return dl.SaveFileTo(ctx, dir, f)
}

// filesFromMessages extracts files from messages slice.
func (*SlackDumper) filesFromMessages(m []Message) []slack.File {
	var files []slack.File

	for i := range m {
		if m[i].Files != nil {
			files = append(files, m[i].Files...)
		}
		// include thread files
		for _, reply := range m[i].ThreadReplies {
			files = append(files, reply.Files...)
		}
	}
	return files
}

type cancelFunc func()

func (sd *SlackDumper) newDownloader(ctx context.Context, dir string, l *rate.Limiter) (ProcessFunc, cancelFunc, error) {
	// set up a file downloader and add it to the post-process functions
	// slice
	dl := downloader.New(
		sd.client,
		downloader.Limiter(l),
		downloader.Retries(sd.options.DownloadRetries),
		downloader.Workers(sd.options.Workers),
	)
	var filesC = make(chan *slack.File, filesCbufSz)

	dlDoneC, err := dl.AsyncDownloader(ctx, dir, filesC)
	if err != nil {
		return nil, nil, err
	}

	fn := func(msg []Message, _ string) (ProcessResult, error) {
		n := sd.pipeFiles(filesC, msg)
		return ProcessResult{Entity: "files", Count: n}, nil
	}

	cancelFn := func() {
		trace.Log(ctx, "info", "closing files channel")
		close(filesC)
		<-dlDoneC
	}
	return fn, cancelFn, nil

}

// pipeFiles scans the messages and sends all the files discovered to the filesC.
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []Message) int {
	// place files in download queue
	fileChunk := sd.filesFromMessages(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
	return len(fileChunk)
}
