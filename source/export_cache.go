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

package source

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

type cacher[K comparable, V any] interface {
	Get(k K) (value V, ok bool)
	GetOrLoad(ctx context.Context, k K, loader func(ctx context.Context, k K) (V, error)) (value V, err error, ok bool)
	Set(k K, v V) (value V, replaced bool)
}

type mapCache[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func newMapCache[K comparable, V any](sz int) *mapCache[K, V] {
	mc := mapCache[K, V]{
		m: make(map[K]V, sz),
	}
	return &mc
}

func (mc *mapCache[K, V]) Get(k K) (V, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	v, ok := mc.m[k]
	if !ok {
		var v V
		return v, false
	}
	return v, true
}

func (mc *mapCache[K, V]) GetOrLoad(ctx context.Context, k K, loader func(ctx context.Context, k K) (V, error)) (V, error, bool) {
	mc.mu.RLock()
	v, ok := mc.m[k]
	mc.mu.RUnlock()
	if ok {
		return v, nil, true
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	v, err := loader(ctx, k)
	if err != nil {
		return v, err, false
	}
	mc.m[k] = v
	return v, nil, false
}

func (mc *mapCache[K, V]) Set(k K, v V) (V, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	old, ok := mc.m[k]
	mc.m[k] = v
	return old, ok
}

// threadCache is a mapping of a channel:thread_id to a list of filenames
type threadCache struct {
	c cacher[string, []string]
}

func newThreadCache(sz int) *threadCache {
	tc := threadCache{
		c: newMapCache[string, []string](sz),
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
