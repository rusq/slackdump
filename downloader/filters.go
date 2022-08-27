package downloader

// fltSeen filters the files from filesC to ensure that no duplicates
// are downloaded.
func (c *Client) fltSeen(filesC <-chan fileRequest) <-chan fileRequest {
	dlQ := make(chan fileRequest)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(dlQ)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[string]bool)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for f := range filesC {
			id := f.File.ID + f.Directory
			if _, ok := seen[id]; ok {
				c.l().Debugf("already seen %q, skipping", Filename(f.File))
				continue
			}
			seen[id] = true
			dlQ <- f
		}
	}()
	return dlQ
}
