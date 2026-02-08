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
package cfg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDir(t *testing.T) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		localDirCache string // set the LocalDirCache to this value
		want          string
	}{
		{
			"returns the UserCacheDir value if global LocalDirCache is empty",
			"",
			filepath.Join(ucd, cacheDirName),
		},
		{
			"returns the LocalDirCache value if it's set",
			"local",
			"local",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := LocalCacheDir
			LocalCacheDir = tt.localDirCache
			t.Cleanup(func() { LocalCacheDir = old })

			if got := CacheDir(); got != tt.want {
				t.Errorf("CacheDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ucd(t *testing.T) {
	type args struct {
		ucdFn func() (string, error)
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"detect OK",
			args{func() (string, error) { return "OK", nil }},
			filepath.Join("OK", cacheDirName),
		},
		{
			"detect failure",
			args{func() (string, error) { return "", errors.New("failed") }},
			".",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ucd(tt.args.ucdFn); got != tt.want {
				t.Errorf("ucd() = %v, want %v", got, tt.want)
			}
		})
	}
}
