package downloader

import (
	"log"

	"github.com/slack-go/slack"
)

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
