package slackdump

import (
	"context"

	"github.com/rusq/slackdump/downloader"
	"github.com/slack-go/slack"
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

// pipeFiles scans the messages and sends all the files discovered to the filesC.
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []Message) int {
	// place files in download queue
	fileChunk := sd.filesFromMessages(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
	return len(fileChunk)
}
