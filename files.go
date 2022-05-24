package slackdump

import (
	"context"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
)

// SaveFileTo saves a single file to the specified directory.
func (sd *SlackDumper) SaveFileTo(ctx context.Context, dir string, f *slack.File) (int64, error) {
	dl := downloader.New(
		sd.client,
		fsadapter.NewDirectory("."),
		downloader.Limiter(network.NewLimiter(network.NoTier, sd.options.Tier3Burst, 0)),
		downloader.Retries(sd.options.DownloadRetries),
		downloader.Workers(sd.options.Workers),
	)
	return dl.SaveFile(ctx, dir, f)
}

// ExtractFiles extracts files from messages slice.
func (*SlackDumper) ExtractFiles(msgs []Message) []slack.File {
	var files []slack.File

	for i := range msgs {
		if msgs[i].Files != nil {
			files = append(files, msgs[i].Files...)
		}
		// include thread files
		for _, reply := range msgs[i].ThreadReplies {
			files = append(files, reply.Files...)
		}
	}
	return files
}

// pipeFiles scans the messages and sends all the files discovered to the filesC.
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []Message) int {
	// place files in the download queue
	fileChunk := sd.ExtractFiles(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
	return len(fileChunk)
}
