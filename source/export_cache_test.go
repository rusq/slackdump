package source

import (
	"context"
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
