package slackdump

import (
	"context"
	"runtime/trace"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/internal/structures/files"
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
		n := pipeFiles(filesC, msg)
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
func pipeFiles(filesC chan<- *slack.File, msgs []types.Message) int {
	// place files in the download queue
	total := 0
	_ = files.Extract(msgs, files.Root, func(file slack.File, _ files.Addr) error {
		filesC <- &file
		total++
		return nil
	})
	return total
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
