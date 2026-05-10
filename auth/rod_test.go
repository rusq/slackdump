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

package auth

import (
	"errors"
	"testing"

	"github.com/rusq/slackauth"
)

func TestRODWithBundledBrowser(t *testing.T) {
	tests := []struct {
		name string
		b    bool
	}{
		{name: "sets true", b: true},
		{name: "sets false", b: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o options
			RODWithBundledBrowser(tt.b)(&o)
			if o.rodOpts.bundledBrowser != tt.b {
				t.Errorf("RODWithBundledBrowser(%v): bundledBrowser = %v, want %v", tt.b, o.rodOpts.bundledBrowser, tt.b)
			}
		})
	}
}

func TestFindInteractiveBrowser(t *testing.T) {
	tests := []struct {
		name   string
		lister func() ([]slackauth.LocalBrowser, error)
		want   string
	}{
		{
			name:   "no browsers found returns empty",
			lister: func() ([]slackauth.LocalBrowser, error) { return nil, slackauth.ErrNoBrowsers },
			want:   "",
		},
		{
			name:   "arbitrary lister error returns empty",
			lister: func() ([]slackauth.LocalBrowser, error) { return nil, errors.New("disk on fire") },
			want:   "",
		},
		{
			name:   "empty slice returns empty",
			lister: func() ([]slackauth.LocalBrowser, error) { return []slackauth.LocalBrowser{}, nil },
			want:   "",
		},
		{
			name: "single chromium returns its path",
			lister: func() ([]slackauth.LocalBrowser, error) {
				return []slackauth.LocalBrowser{{Name: "Chromium", Path: "/usr/bin/chromium"}}, nil
			},
			want: "/usr/bin/chromium",
		},
		{
			name: "multiple browsers picks the first (slackauth-prioritised) one",
			lister: func() ([]slackauth.LocalBrowser, error) {
				return []slackauth.LocalBrowser{
					{Name: "Microsoft Edge", Path: "/usr/bin/microsoft-edge"},
					{Name: "Chromium", Path: "/usr/bin/chromium"},
				}, nil
			},
			want: "/usr/bin/microsoft-edge",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findInteractiveBrowser(tt.lister); got != tt.want {
				t.Errorf("findInteractiveBrowser() = %q, want %q", got, tt.want)
			}
		})
	}
}
