package slackdump

import (
	"context"
	"runtime/trace"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/types"
)

const (
	// files channel buffer size. I don't know, i just like 20, doesn't really matter.
	filesCbufSz = 20
)

// ProcessFunc is the signature of the function Dump* functions accept and call for each
// API call result.
type ProcessFunc func(msg []types.Message, channelID string) (ProcessResult, error)

// cancelFunc may be returned by some process function constructors.
type cancelFunc func()

// runProcessFuncs runs processFn sequentially and return results of execution.
func runProcessFuncs(m []types.Message, channelID string, processFn ...ProcessFunc) (ProcessResults, error) {
	var prs ProcessResults
	for _, fn := range processFn {
		res, err := fn(m, channelID)
		if err != nil {
			return nil, err
		}
		prs = append(prs, res)
	}
	return prs, nil
}

// newFileProcessFn returns a file process function that will save the conversation files to
// directory dir, rate limited by limiter l.  It returns ProcessFunction and CancelFunc.  CancelFunc
// must be called, i.e. by deferring it's execution.
func (sd *SlackDumper) newFileProcessFn(ctx context.Context, dir string, l *rate.Limiter) (ProcessFunc, cancelFunc, error) {
	// set up a file downloader and add it to the post-process functions
	// slice
	dl := downloader.New(
		sd.client,
		sd.fs,
		downloader.Limiter(l),
		downloader.Retries(sd.options.DownloadRetries),
		downloader.Workers(sd.options.Workers),
	)
	var filesC = make(chan *slack.File, filesCbufSz)

	dlDoneC, err := dl.AsyncDownloader(ctx, dir, filesC)
	if err != nil {
		return nil, nil, err
	}

	fn := func(msg []types.Message, _ string) (ProcessResult, error) {
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

// ExtractFiles extracts files from messages slice.
func (*SlackDumper) ExtractFiles(msgs []types.Message) []slack.File {
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
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []types.Message) int {
	// place files in the download queue
	fileChunk := sd.ExtractFiles(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
	return len(fileChunk)
}

// newThreadProcessFn returns the new thread processor function.  It will use limiter l
// to limit the API calls rate.
func (sd *SlackDumper) newThreadProcessFn(ctx context.Context, l *rate.Limiter) ProcessFunc {
	processFn := func(chunk []types.Message, channelID string) (ProcessResult, error) {
		n, err := sd.populateThreads(ctx, l, chunk, channelID, sd.dumpThread)
		if err != nil {
			return ProcessResult{}, err
		}
		return ProcessResult{Entity: "threads", Count: n}, nil
	}
	return processFn
}
