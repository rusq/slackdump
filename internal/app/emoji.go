package app

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

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/logger"
)

const numworkers = 12

// Emoji saves all emojis to "emoji" subdirectory.
func Emoji(ctx context.Context, cfg Config, prov auth.Provider) error {
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.Options)
	if err != nil {
		return err
	}
	fsa, err := fsadapter.ForFilename(cfg.Output.Base)
	if err != nil {
		return fmt.Errorf("unable to initialise adapter for %s: %w", cfg.Output.Base, err)
	}
	defer fsadapter.Close(fsa)

	emojis, err := sess.DumpEmojis(ctx)
	if err != nil {
		return fmt.Errorf("error during emoji dump: %w", err)
	}
	bIndex, err := json.Marshal(emojis)
	if err != nil {
		return fmt.Errorf("failed marshalling emoji index: %w", err)
	}
	if err := fsa.WriteFile("index.json", bIndex, 0644); err != nil {
		return fmt.Errorf("failed writing emoji index: %w", err)
	}

	return saveEmojis(ctx, fsa, emojis)
}

func saveEmojis(ctx context.Context, fsa fsadapter.FS, emojis map[string]string) error {

	var (
		emojiC  = make(chan emoji)
		resultC = make(chan emojiResult)
	)

	go func() {
		defer close(emojiC)
		for name, uri := range emojis {
			select {
			case <-ctx.Done():
				return
			case emojiC <- emoji{name, uri}:
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < numworkers; i++ {
		wg.Add(1)
		go func() {
			emojiWorker(ctx, fsa, emojiC, resultC)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(resultC)
	}()

	var (
		total = len(emojis)
		count = 0
	)
	for res := range resultC {
		if res.err != nil {
			if errors.Is(res.err, context.Canceled) {
				return res.err
			}
			logger.Default.Printf("failed: %q: %s", res.name, res.err)
			continue
		}
		count++
		logger.Default.Printf("downloaded % 5d/%d %q", count, total, res.name)
	}

	return nil
}

type emoji [2]string

type emojiResult struct {
	name string
	err  error
}

func emojiWorker(ctx context.Context, fsa fsadapter.FS, emojiC <-chan emoji, resultC chan<- emojiResult) {
	for {
		select {
		case <-ctx.Done():
			resultC <- emojiResult{err: ctx.Err()}
			return
		case emoji, more := <-emojiC:
			if !more {
				return
			}
			if strings.HasPrefix(emoji[1], "alias:") {
				resultC <- emojiResult{name: emoji[0] + "(alias, skipped)"}
				break
			}
			err := emojiDownload(ctx, fsa, emoji[0], emoji[1])
			resultC <- emojiResult{name: emoji[0], err: err}
		}
	}
}

func emojiDownload(ctx context.Context, fsa fsadapter.FS, name, uri string) error {
	// req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	// if err != nil {
	// 	return err
	// }
	// resp, err := http.DefaultClient.Do(req)
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	savepath := path.Join("emojis", name+".png")
	wc, err := fsa.Create(savepath)
	if err != nil {
		return err
	}
	defer wc.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("server NOT OK")
	}
	if _, err := io.Copy(wc, resp.Body); err != nil {
		return err
	}
	return nil
}
