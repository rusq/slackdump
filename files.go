package slackdump

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

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
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []Message) {
	if !sd.options.DumpFiles {
		return
	}
	// place files in download queue
	fileChunk := sd.filesFromMessages(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
}

// SaveFileTo saves a single file to the specified directory.
func (sd *SlackDumper) SaveFileTo(ctx context.Context, dir string, f *slack.File) (int64, error) {
	return sd.saveFileWithLimiter(ctx, newLimiter(noTier, sd.options.Tier3Burst, 0), dir, f)
}

// saveFileWithLimiter saves the file to specified directory, it will use the provided limiter l for throttling.
func (sd *SlackDumper) saveFileWithLimiter(ctx context.Context, l *rate.Limiter, dir string, sf *slack.File) (int64, error) {
	filePath := filepath.Join(dir, filename(sf))
	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if err := withRetry(ctx, l, sd.options.DownloadRetries, func() error {
		region := trace.StartRegion(ctx, "GetFile")
		defer region.End()

		if err := sd.client.GetFile(sf.URLPrivateDownload, f); err != nil {
			// cleanup if download failed.
			f.Close()
			if e := os.RemoveAll(filePath); e != nil {
				trace.Logf(ctx, "error", "removing file after unsuccesful download failed with: %s", e)
			}
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return int64(sf.Size), nil
}

// filename returns name of the file
func filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

// newFileDownloader starts sd.Options.worker goroutines to download files in
// parallel. It will download any file that is received on toDownload channel. It
// returns the "done" channel and an error, the "done" channel will be closed
// once all downloads are completed.
func (sd *SlackDumper) newFileDownloader(ctx context.Context, l *rate.Limiter, dir string, toDownload <-chan *slack.File) (chan struct{}, error) {
	done := make(chan struct{})

	if !sd.options.DumpFiles {
		// terminating if DumpFiles is not enabled.
		close(done)
		return done, nil
	}

	if dir == "" {
		return nil, errors.New("empty directory")
	}

	if err := os.Mkdir(dir, 0777); err != nil {
		if !os.IsExist(err) {
			close(done)
			return done, err
		}
	}

	var wg sync.WaitGroup
	// create workers
	for i := 0; i < sd.options.Workers; i++ {
		wg.Add(1)
		go func() {
			sd.worker(ctx, l, dir, seenFilter(toDownload))
			wg.Done()
		}()
	}

	// sentinel
	go func() {
		wg.Wait()
		close(done)
	}()

	return done, nil
}

func (sd *SlackDumper) worker(ctx context.Context, l *rate.Limiter, dir string, filesC <-chan *slack.File) {
	for {
		select {
		case <-ctx.Done():
			trace.Log(ctx, "warn", "worker context cancelled")
			return
		case file, moar := <-filesC:
			if !moar {
				return
			}
			dlog.Debugf("saving %s, size: %d", filename(file), file.Size)
			n, err := sd.saveFileWithLimiter(ctx, l, dir, file)
			if err != nil {
				dlog.Printf("error saving %q: %s", filename(file), err)
				break
			}
			dlog.Printf("file %s saved: %d bytes written", filename(file), n)
		}
	}
}

// seenFilter filters the files from filesC to ensure that no duplicates
// are downloaded.
func seenFilter(filesC <-chan *slack.File) <-chan *slack.File {
	dlQ := make(chan *slack.File)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(dlQ)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[string]bool)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for f := range filesC {
			if _, ok := seen[f.ID]; ok {
				log.Printf("already seen %s, skipping", filename(f))
				continue
			}
			seen[f.ID] = true
			dlQ <- f
		}
	}()
	return dlQ
}
