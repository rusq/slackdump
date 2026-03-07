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

// Package emojidl provides functions to dump the all slack emojis for a
// workspace. It skips the "alias" emojis, so only original emoji with an
// original name is present. If you need to find the alias - look it up in the
// index.json. The directory structure is the following:
//
//	.
//	+- emojis
//	|  +- foo.png
//	|  +- bar.png
//	:  :
//	|  +- baz.png
//	+- index.json
//
// Where index.json contains the emoji index, and *.png files under emojis
// directory are individual emojis.
package emojidl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"sync"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/edge"
)

//go:generate mockgen -source emoji.go -destination emoji_mock_test.go -package emojidl
type EdgeEmojiLister interface {
	AdminEmojiList(ctx context.Context) iter.Seq2[edge.EmojiResult, error]
}

type StatusFunc func(name string, total, count int)

// DlEdgeFS downloads the emojis and saves them to the fsa. It spawns numWorker
// goroutines for getting the files. It will call fetchFn for each emoji.
func DlEdgeFS(ctx context.Context, sess EdgeEmojiLister, fsa fsadapter.FS, opt *Options, cb StatusFunc) error {
	if opt == nil {
		opt = &Options{}
	}
	lg := cfg.Log
	lg.DebugContext(ctx, "startup params", "dir", emojiDir, "numWorkers", numWorkers, "failFast", opt.FailFast)
	if cb == nil {
		cb = func(name string, total, count int) {}
	}

	var (
		emojiC  = make(chan edge.Emoji)
		totalC  = make(chan int)
		genErrC = make(chan error)
		resultC = make(chan result)
	)

	// Async download pipeline.
	workerFn := nofetchworker
	if opt.WithFiles {
		workerFn = worker
	}

	// 1. generator, send emojis into the emojiC channel.
	go func() {
		var once sync.Once
		defer close(totalC)
		defer close(emojiC)

		for chunk, err := range sess.AdminEmojiList(ctx) {
			if err != nil {
				genErrC <- err
				return
			}
			lg.DebugContext(ctx, "got emojis", "count", len(chunk.Emoji), "disabled", len(chunk.DisabledEmoji), "total", chunk.Total)
			once.Do(func() { totalC <- chunk.Total }) // send total count once.
			for _, emoji := range chunk.Emoji {
				select {
				case <-ctx.Done():
					return
				case emojiC <- emoji:
				}
			}
		}
	}()

	// 2. Download workers, download the emojis.
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			workerFn(ctx, fsa, emojiC, resultC)
			wg.Done()
		}()
	}
	// 3. Sentinel, closes the result channel once all workers are finished.
	go func() {
		wg.Wait()
		close(resultC)
	}()

	// 4. Result processor, receives download results and logs any errors that
	//    may have occurred.
	var (
		count = 0
		total = <-totalC // if there's a generator error, this will receive 0.
	)
	emojis := make(map[string]edge.Emoji, total)
LOOP:
	for {
		select {
		case genErr := <-genErrC:
			// generator error.
			if genErr != nil {
				return fmt.Errorf("failed to get emoji list: %w", genErr)
			}
		case res, more := <-resultC:
			if !more {
				break LOOP
			}
			lg := lg.With("name", res.emoji.Name)
			if res.err != nil {
				if errors.Is(res.err, context.Canceled) {
					return res.err
				}
				if opt.FailFast {
					return fmt.Errorf("failed: %q: %w", res.emoji.Name, res.err)
				}
				lg.WarnContext(ctx, "failed", "error", res.err)
			}
			emojis[res.emoji.Name] = res.emoji // to resemble the legacy code.
			count++
			cb(res.emoji.Name, total, count)
		}
	}
	out, err := fsa.Create("index.json")
	if err != nil {
		return err
	}
	defer out.Close()
	if err := json.NewEncoder(out).Encode(emojis); err != nil {
		return err
	}

	return nil
}

type result struct {
	emoji   edge.Emoji
	skipped bool
	err     error
}

// worker is the function that runs in a separate goroutine and downloads emoji
// received from emojiC. The result of the operation is sent to resultC channel.
// fn is called for each received emoji.
func worker(ctx context.Context, fsa fsadapter.FS, emojiC <-chan edge.Emoji, resultC chan<- result) {
	for {
		select {
		case <-ctx.Done():
			resultC <- result{err: ctx.Err()}
			return
		case em, more := <-emojiC:
			if !more {
				return
			}
			if em.IsAlias != 0 {
				resultC <- result{emoji: em, skipped: true}
				break
			}
			err := fetchFn(ctx, fsa, emojiDir, em.Name, em.URL)
			resultC <- result{emoji: em, err: err}
		}
	}
}

func nofetchworker(ctx context.Context, _ fsadapter.FS, emojiC <-chan edge.Emoji, resultC chan<- result) {
	for em := range emojiC {
		resultC <- result{emoji: em}
	}
}
