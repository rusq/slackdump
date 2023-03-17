package slackdump

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/types"
)

func Test_usercache_get(t *testing.T) {
	type args struct {
		retention time.Duration
	}
	tests := []struct {
		name    string
		cache   *usercache
		args    args
		want    types.Users
		wantErr bool
	}{
		{
			name: "empty cache",
			cache: &usercache{
				users:    nil,
				mu:       sync.RWMutex{},
				cachedAt: time.Time{},
			},
			args: args{
				retention: time.Hour,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "expired cache",
			cache: &usercache{
				users:    testUsers,
				mu:       sync.RWMutex{},
				cachedAt: time.Now().Add(-time.Hour),
			},
			args: args{
				retention: time.Minute,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid cache",
			cache: &usercache{
				users:    testUsers,
				mu:       sync.RWMutex{},
				cachedAt: time.Now(),
			},
			args: args{
				retention: time.Hour,
			},
			want:    testUsers,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cache.get(tt.args.retention)
			if (err != nil) != tt.wantErr {
				t.Errorf("usercache.get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("usercache.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_usercache_set(t *testing.T) {
	type args struct {
		users types.Users
	}
	tests := []struct {
		name           string
		cache          *usercache
		args           args
		wantCacheUsers types.Users
	}{
		{
			name: "set cache",
			cache: &usercache{
				users:    nil,
				mu:       sync.RWMutex{},
				cachedAt: time.Time{},
			},
			args: args{
				users: testUsers,
			},
			wantCacheUsers: testUsers,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cache.set(tt.args.users)
		})
	}
}
