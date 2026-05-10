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

package wspcfg

import (
	"flag"
	"testing"
	"time"

	"github.com/rusq/slackdump/v4/auth"
)

func TestSetWspFlags_bundledBrowser(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "default is false", args: []string{}, want: false},
		{name: "-bundled-browser sets true", args: []string{"-bundled-browser"}, want: true},
		{name: "-bundled-browser=true", args: []string{"-bundled-browser=true"}, want: true},
		{name: "-bundled-browser=false", args: []string{"-bundled-browser=false"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BundledBrowser = false
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			SetWspFlags(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if BundledBrowser != tt.want {
				t.Errorf("BundledBrowser = %v, want %v", BundledBrowser, tt.want)
			}
		})
	}
}

func TestRodAuthOptions_WiresBundledBrowserPolarity(t *testing.T) {
	origHeadless := rodWithHeadlessTimeout
	origUA := rodWithUserAgent
	origBundled := rodWithBundledBrowser
	origAuto := rodWithInteractiveAuto
	defer func() {
		rodWithHeadlessTimeout = origHeadless
		rodWithUserAgent = origUA
		rodWithBundledBrowser = origBundled
		rodWithInteractiveAuto = origAuto
	}()

	var (
		gotHeadless time.Duration
		gotUA       string
		gotBundled  bool
		gotAuto     bool
	)
	rodWithHeadlessTimeout = func(d time.Duration) auth.Option {
		gotHeadless = d
		return nil
	}
	rodWithUserAgent = func(s string) auth.Option {
		gotUA = s
		return nil
	}
	rodWithBundledBrowser = func(b bool) auth.Option {
		gotBundled = b
		return nil
	}
	rodWithInteractiveAuto = func(b bool) auth.Option {
		gotAuto = b
		return nil
	}

	HeadlessTimeout = 77 * time.Second
	RODUserAgent = "ua-test"
	BundledBrowser = true
	opts := RodAuthOptions()

	if len(opts) != 4 {
		t.Fatalf("RodAuthOptions() len = %d, want 4", len(opts))
	}
	if gotHeadless != 77*time.Second {
		t.Errorf("headless timeout = %v, want %v", gotHeadless, 77*time.Second)
	}
	if gotUA != "ua-test" {
		t.Errorf("user agent = %q, want %q", gotUA, "ua-test")
	}
	if !gotBundled {
		t.Errorf("bundled browser = %v, want true", gotBundled)
	}
	if gotAuto {
		t.Errorf("interactive auto = %v, want false", gotAuto)
	}
}
