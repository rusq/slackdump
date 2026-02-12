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
	"testing"

	"github.com/stretchr/testify/assert"
)

const testCacheSz = 100

func Test_threadCache_Exists(t *testing.T) {
	type args struct {
		chanName string
	}
	tests := []struct {
		name   string
		fields *threadCache
		args   args
		want   bool
	}{
		{
			name:   "does not exist",
			fields: newThreadCache(testCacheSz),
			args:   args{"general"},
			want:   false,
		},
		{
			name: "exists",
			fields: func() *threadCache {
				tc := newThreadCache(testCacheSz)
				tc.c.Set("general", []string{})
				return tc
			}(),
			args: args{"general"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := tt.fields
			if got := tc.Exists(tt.args.chanName); got != tt.want {
				t.Errorf("threadCache.Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestCache() *threadCache {
	return newThreadCache(testCacheSz)
}

func assertChannelExists(t *testing.T, tc *threadCache, chanName string) {
	t.Helper()
	cn, ok := tc.c.Get(chanName)
	if !ok {
		t.Errorf("expected the channel key %q to be added as well", chanName)
	}
	assert.Equal(t, []string{}, cn)
}

func assertThreadFilesEq(t *testing.T, tc *threadCache, chanName string, threadTS string, expected []string) {
	t.Helper()
	v, ok := tc.c.Get(cacheKey(chanName, threadTS))
	if !ok {
		t.Fatal("expected the value to be in the cache")
	}
	assert.Equal(t, expected, v)
}

func Test_threadCache_Update(t *testing.T) {
	type args struct {
		ctx      context.Context
		chanName string
		threadTS string
		filename string
	}
	tests := []struct {
		name         string
		fields       *threadCache
		args         args
		checkCacheFn func(t *testing.T, tc *threadCache)
		wantErr      bool
	}{
		{
			name:   "new item added",
			fields: newTestCache(),
			args: args{
				ctx:      t.Context(),
				chanName: "general",
				threadTS: "12345.6789",
				filename: "2011-09-16.json",
			},
			checkCacheFn: func(t *testing.T, tc *threadCache) {
				assertChannelExists(t, tc, "general")
				assertThreadFilesEq(t, tc, "general", "12345.6789", []string{"2011-09-16.json"})

			},
			wantErr: false,
		},
		{
			name: "adding file to existing thread",
			fields: func() *threadCache {
				tc := newTestCache()
				tc.c.Set(cacheKey("general", "1000.2000"), []string{"initial.json"})
				return tc
			}(),
			args: args{
				ctx:      t.Context(),
				chanName: "general",
				threadTS: "1000.2000",
				filename: "unittest.json",
			},
			checkCacheFn: func(t *testing.T, tc *threadCache) {
				assertChannelExists(t, tc, "general")
				assertThreadFilesEq(t, tc, "general", "1000.2000", []string{
					"initial.json",
					"unittest.json",
				})
			},
			wantErr: false,
		},
		{
			name: "adding another thread",
			fields: func() *threadCache {
				tc := newTestCache()
				tc.c.Set(cacheKey("general", "1000.2000"), []string{"initial.json"})
				return tc
			}(),
			args: args{
				ctx:      t.Context(),
				chanName: "general",
				threadTS: "3000.4000",
				filename: "unittest.json",
			},
			checkCacheFn: func(t *testing.T, tc *threadCache) {
				assertChannelExists(t, tc, "general")
				assertThreadFilesEq(t, tc, "general", "1000.2000", []string{
					"initial.json",
				})
				assertThreadFilesEq(t, tc, "general", "3000.4000", []string{
					"unittest.json",
				})
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := tt.fields
			if err := tc.Update(tt.args.ctx, tt.args.chanName, tt.args.threadTS, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("threadCache.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.checkCacheFn(t, tc)
		})
	}
}

func Test_mapCache_Get(t *testing.T) {
	tests := []struct {
		name  string
		cache *mapCache[string, int]
		k     string
		want  int
		want2 bool
	}{
		{
			name: "gets an existing value",
			cache: &mapCache[string, int]{
				m: map[string]int{"test": 42},
			},
			k:     "test",
			want:  42,
			want2: true,
		},
		{
			name: "value does not exist",
			cache: &mapCache[string, int]{
				m: map[string]int{"test": 42},
			},
			k:     "other",
			want:  0,
			want2: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mc = tt.cache
			got, got2 := mc.Get(tt.k)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want2, got2)
		})
	}
}

func Test_mapCache_Set(t *testing.T) {
	tests := []struct {
		name  string
		cache *mapCache[string, int]
		k     string
		v     int
		want  int
		want2 bool
	}{
		{
			name: "sets the value",
			cache: &mapCache[string, int]{
				m: map[string]int{},
			},
			k:     "test",
			v:     42,
			want:  0,
			want2: false,
		},
		{
			name: "replaces the value",
			cache: &mapCache[string, int]{
				m: map[string]int{"test": 42},
			},
			k:     "test",
			v:     24,
			want:  42,
			want2: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mc = tt.cache
			got, got2 := mc.Set(tt.k, tt.v)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want2, got2)
		})
	}
}

func Test_mapCache_GetOrLoad(t *testing.T) {
	tests := []struct {
		name   string // description of this test case
		cache  *mapCache[string, int]
		k      string
		loader func(ctx context.Context, k string) (int, error)
		want   int
		want2  error
		want3  bool
	}{
		{
			name: "already in the map",
			cache: &mapCache[string, int]{
				m: map[string]int{"test": 42},
			},
			k: "test",
			loader: func(ctx context.Context, k string) (int, error) {
				panic("should not be called")
			},
			want:  42,
			want2: nil,
			want3: true,
		},
		{
			name: "not in the map, load called",
			cache: &mapCache[string, int]{
				m: map[string]int{},
			},
			k: "get_me",
			loader: func(ctx context.Context, k string) (int, error) {
				return 100, nil
			},
			want:  100,
			want2: nil,
			want3: false,
		},
		{
			name: "load fails",
			cache: &mapCache[string, int]{
				m: map[string]int{},
			},
			k: "fails",
			loader: func(ctx context.Context, k string) (int, error) {
				return 0, errors.New("error")
			},
			want:  0,
			want2: errors.New("error"),
			want3: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var mc = tt.cache
			got, got2, got3 := mc.GetOrLoad(t.Context(), tt.k, tt.loader)
			// TODO: update the condition below to compare got with tt.want.
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want2, got2)
			assert.Equal(t, tt.want3, got3)
		})
	}
}
