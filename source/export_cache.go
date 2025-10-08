package source

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/phuslu/lru"
)

// threadCache is a mapping of a channel:thread_id to a list of filenames
type threadCache struct {
	c *lru.LRUCache[string, []string]
}

func newThreadCache(sz int) *threadCache {
	tc := threadCache{
		c: lru.NewLRUCache[string, []string](sz),
	}
	return &tc
}

// Exists returns true if channel with chanName exists in cache.
func (tc *threadCache) Exists(chanName string) bool {
	_, exists := tc.c.Get(chanName)
	return exists
}

func (tc *threadCache) Update(ctx context.Context, chanName string, threadTS string, filename string) error {
	// we add an entry with a channel name as a key to indicate that the thread
	// information for the channel is cached.
	_, _, _ = tc.c.GetOrLoad(ctx, chanName, func(ctx context.Context, s string) ([]string, error) {
		return []string{}, nil
	})

	threadKey := cacheKey(chanName, threadTS)
	files, err, ok := tc.c.GetOrLoad(ctx, threadKey, func(context.Context, string) ([]string, error) {
		return []string{filename}, nil
	})
	if err != nil {
		return fmt.Errorf("unexpected cache error: %w", err)
	}
	if !ok {
		// value was not in cache, and we have already added the filename in the loader function.
		return nil
	}
	if slices.Contains(files, filename) {
		// file is already in the cache.
		return nil
	}
	files = append(files, filename)

	_, replaced := tc.c.Set(threadKey, files)
	if !replaced {
		return errors.New("internal error: expected to replace the value, but did nothing")
	}
	return nil
}

func (tc *threadCache) Get(chanName string, threadTS string) ([]string, bool) {
	return tc.c.Get(cacheKey(chanName, threadTS))
}

func cacheKey(chanName string, threadTS string) string {
	return chanName + "/" + threadTS
}
