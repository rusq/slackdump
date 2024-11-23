// Package emojidl provides functions to dump the all slack emojis for a workspace.
// It skips the "alias" emojis, so only original an emoji with an original name
// is present. If you need to find the alias - lookup the index.json. The
// directory structure is the following:
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
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/edge"
)

//go:generate mockgen -source emoji.go -destination emoji_mock_test.go -package emojidl
type EdgeEmojiLister interface {
	AdminEmojiList(ctx context.Context) iter.Seq2[edge.EmojiResult, error]
}

// DlEdgeFS downloads the emojis and saves them to the fsa. It spawns numWorker
// goroutines for getting the files. It will call fetchFn for each emoji.
func DlEdgeFS(ctx context.Context, sess EdgeEmojiLister, fsa fsadapter.FS, failFast bool) error {
	lg := cfg.Log.With("in", "fetch", "dir", emojiDir, "numWorkers", numWorkers, "failFast", failFast)

	var (
		emojiC  = make(chan edge.Emoji)
		totalC  = make(chan int)
		genErrC = make(chan error)
		resultC = make(chan edgeResult)
	)

	// Async download pipeline.

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
			worker2(ctx, fsa, emojiC, resultC)
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
		total = <-totalC
	)
	var emojis = make(map[string]edge.Emoji, total)
LOOP:
	for {
		select {

		case genErr := <-genErrC:
			if genErr != nil {
				return fmt.Errorf("failed to get emoji list: %w", genErr)
			}
		case res, more := <-resultC:
			if !more {
				break LOOP
			}
			if res.err != nil {
				if errors.Is(res.err, context.Canceled) {
					return res.err
				}
				if failFast {
					return fmt.Errorf("failed: %q: %w", res.emoji.Name, res.err)
				}
				lg.WarnContext(ctx, "failed", "name", res.emoji.Name, "error", res.err)
			}
			emojis[res.emoji.Name] = res.emoji // to resemble the legacy code.
			count++
			lg.InfoContext(ctx, "downloaded", "count", count, "total", total, "name", res.emoji.Name)
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

type edgeResult struct {
	emoji   edge.Emoji
	skipped bool
	err     error
}

// worker is the function that runs in a separate goroutine and downloads emoji
// received from emojiC. The result of the operation is sent to resultC channel.
// fn is called for each received emoji.
func worker2(ctx context.Context, fsa fsadapter.FS, emojiC <-chan edge.Emoji, resultC chan<- edgeResult) {
	for {
		select {
		case <-ctx.Done():
			resultC <- edgeResult{err: ctx.Err()}
			return
		case em, more := <-emojiC:
			if !more {
				return
			}
			if em.IsAlias != 0 {
				resultC <- edgeResult{emoji: em, skipped: true}
				break
			}
			err := fetchFn(ctx, fsa, emojiDir, em.Name, em.URL)
			resultC <- edgeResult{emoji: em, err: err}
		}
	}
}
