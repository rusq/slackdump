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
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/edge"
)

const (
	numWorkers = 12       // default number of download workers.
	emojiDir   = "emojis" // directory where all emojis are downloaded.
)

var fetchFn = fetchEmoji

//go:generate mockgen -source emoji.go -destination emoji_mock_test.go -package emojidl
type EmojiDumper interface {
	DumpEmojis(ctx context.Context) (map[string]string, error)
}

type Options struct {
	FailFast   bool
	NoDownload bool
}

// DlFS downloads all emojis from the workspace and saves them to the fsa.
func DlFS(ctx context.Context, sess EmojiDumper, fsa fsadapter.FS, opt *Options, cb StatusFunc) error {
	if opt == nil {
		opt = &Options{}
	}
	emojis, err := sess.DumpEmojis(ctx)
	if err != nil {
		return fmt.Errorf("error during emoji dump: %w", err)
	}

	bIndex, err := json.Marshal(emojis)
	if err != nil {
		return fmt.Errorf("error marshalling emoji index: %w", err)
	}
	if err := fsa.WriteFile("index.json", bIndex, 0o644); err != nil {
		return fmt.Errorf("failed writing emoji index: %w", err)
	}

	if opt.NoDownload {
		return nil
	} else {
		if err := fetch(ctx, fsa, emojis, opt.FailFast, cb); err != nil {
			return fmt.Errorf("failed downloading emojis: %w", err)
		}
	}
	return nil
}

func ift[T any](cond bool, t, f T) T {
	if cond {
		return t
	}
	return f
}

// fetch downloads the emojis and saves them to the fsa. It spawns numWorker
// goroutines for getting the files. It will call fetchFn for each emoji.
func fetch(ctx context.Context, fsa fsadapter.FS, emojis map[string]string, failFast bool, cb StatusFunc) error {
	lg := cfg.Log
	lg.DebugContext(ctx, "startup params", "dir", emojiDir, "numWorkers", numWorkers, "failFast", failFast)

	if cb == nil {
		cb = func(name string, total, count int) {}
	}

	var (
		emojiC  = make(chan edge.Emoji)
		resultC = make(chan result)
	)

	const (
		aliasPrefix = "alias:"
		aliasLen    = len(aliasPrefix)
	)

	// Async download pipeline.

	// 1. generator, send emojis into the emojiC channel.
	go func() {
		defer close(emojiC)

		for name, uri := range emojis {
			isAlias := strings.HasPrefix(uri, aliasPrefix)
			emoji := edge.Emoji{
				Name:     name,
				URL:      uri,
				IsAlias:  ift(isAlias, 1, 0),
				AliasFor: ift(isAlias, uri[aliasLen:], ""),
			}
			select {
			case <-ctx.Done():
				return
			case emojiC <- emoji:
			}
		}
	}()

	// 2. Download workers, download the emojis.
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			worker(ctx, fsa, emojiC, resultC)
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
		total = len(emojis)
		count = 0
	)
	lg = lg.With("total", total)
	for res := range resultC {
		lg := lg.With("name", res.emoji.Name)
		if res.err != nil {
			if errors.Is(res.err, context.Canceled) {
				return res.err
			}
			if failFast {
				return fmt.Errorf("failed: %q: %w", res.emoji.Name, res.err)
			}
			lg.WarnContext(ctx, "failed", "error", res.err)
		}
		count++
		cb(res.emoji.Name, total, count)
	}

	return nil
}

// fetchEmoji downloads one emoji file from uri into the filename dir/name.png
// within the filesystem adapter fsa.
func fetchEmoji(ctx context.Context, fsa fsadapter.FS, dir string, name, uri string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	emojiExt := path.Ext(uri) // get the extension from the uri.
	filename := path.Join(dir, name+emojiExt)
	wc, err := fsa.Create(filename)
	if err != nil {
		return err
	}
	defer wc.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid server status code: %d (%s)", resp.StatusCode, resp.Status)
	}

	if _, err := io.Copy(wc, resp.Body); err != nil {
		return err
	}

	return nil
}
