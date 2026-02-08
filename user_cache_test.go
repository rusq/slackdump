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
package slackdump

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rusq/slackdump/v4/types"
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
