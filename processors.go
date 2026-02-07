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
	"context"
	"errors"
	"fmt"
	"path"
	"runtime/trace"
	"strings"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/downloader"
	"github.com/rusq/slackdump/v3/internal/structures/files"
	"github.com/rusq/slackdump/v3/types"
)

const (
	// files channel buffer size. I don't know, i just like 20, doesn't really matter.
	filesCbufSz = 20
)

// ProcessFunc is the signature of the processor function that Dump* family
// functions accept and call for each API call result.  It can be used to
// modify in-place the slice of messages, returned from API, before they are
// appended to the slice that will be returned by Dump*.  Messages passed to
// this function are unsorted.
type ProcessFunc func(msg []types.Message, channelID string) (ProcessResult, error)

// ProcessResult contains the result of processing.
type ProcessResult struct {
	// Entity is the type of entity that this result is related to.
	Entity string
	// Count is the number of items processed during processing.
	Count int
}

func (pr ProcessResult) String() string {
	return fmt.Sprintf("%s: %d", pr.Entity, pr.Count)
}

// ProcessResults is the slice of ProcessResult
type ProcessResults []ProcessResult

func (prs ProcessResults) String() string {
	var results []string
	for _, res := range prs {
		results = append(results, res.String())
	}
	return strings.Join(results, ", ")
}

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

// newFileProcessFn returns a file process function that will save the
// conversation files to directory dir on the slackdump filesystem, rate limited
// by limiter l.  The File.PublicURL will be updated to point to the downloaded
// file, instead of Slack server URL.  It returns ProcessFunction and
// CancelFunc. CancelFunc must be called, i.e. by deferring it's execution.
func (s *Session) newFileProcessFn(ctx context.Context, dir string, l *rate.Limiter) (ProcessFunc, cancelFunc, error) {
	if s.fs == nil {
		return nil, nil, errors.New("filesystem not set, unable to download files")
	}
	dl := downloader.New(s.client, s.fs, downloader.Limiter(l), downloader.Retries(s.cfg.limits.DownloadRetries), downloader.Workers(s.cfg.limits.Workers))
	// set up a file downloader and add it to the post-process functions
	// slice
	fileRequests := make(chan downloader.Request, filesCbufSz)

	dlDoneC, err := dl.AsyncDownloader(ctx, fileRequests)
	if err != nil {
		return nil, nil, err
	}

	fn := func(msg []types.Message, _ string) (ProcessResult, error) {
		n := pipeAndUpdateFiles(fileRequests, msg, dir)
		return ProcessResult{Entity: "files", Count: n}, nil
	}

	cancelFn := func() {
		trace.Log(ctx, "info", "closing files channel")
		close(fileRequests)
		<-dlDoneC
	}
	return fn, cancelFn, nil
}

// pipeAndUpdateFiles scans the messages and sends all the files discovered to
// the filesC.
func pipeAndUpdateFiles(filesC chan<- downloader.Request, msgs []types.Message, dir string) int {
	// place files in the download queue
	total := 0
	_ = files.Extract(msgs, files.Root, func(file slack.File, addr files.Addr) error {
		filesC <- downloader.Request{
			Fullpath: path.Join(dir, downloader.Filename(&file)),
			URL:      file.URLPrivateDownload,
		}
		total++
		return files.Update(msgs, addr, files.UpdatePathFn(path.Join(dir, downloader.Filename(&file))))
	})
	return total
}

// newThreadProcessFn returns the new thread processor function.  It will use limiter l
// to limit the API calls rate.
func (s *Session) newThreadProcessFn(ctx context.Context, l *rate.Limiter, oldest, latest time.Time) ProcessFunc {
	processFn := func(chunk []types.Message, channelID string) (ProcessResult, error) {
		n, err := s.populateThreads(ctx, l, chunk, channelID, oldest, latest, s.dumpThread)
		if err != nil {
			return ProcessResult{}, err
		}
		return ProcessResult{Entity: "threads", Count: n}, nil
	}
	return processFn
}
