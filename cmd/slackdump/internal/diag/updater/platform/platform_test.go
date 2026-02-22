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

package platform

import (
	"context"
	"testing"
)

func TestDetect(t *testing.T) {
	ctx := context.Background()
	p, err := Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if p.OS == "" {
		t.Error("OS should not be empty")
	}
	if p.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if p.ExePath == "" {
		t.Error("ExePath should not be empty")
	}
	if p.PackageSystem == Unknown {
		t.Error("PackageSystem should be detected")
	}

	t.Logf("Detected platform: OS=%s, Arch=%s, PackageSystem=%s, ExePath=%s",
		p.OS, p.Arch, p.PackageSystem, p.ExePath)
}

func TestPackageSystem_String(t *testing.T) {
	tests := []struct {
		name string
		p    PackageSystem
		want string
	}{
		{"Unknown", Unknown, "unknown"},
		{"Homebrew", Homebrew, "homebrew"},
		{"Pacman", Pacman, "pacman"},
		{"APT", APT, "apt"},
		{"Binary", Binary, "binary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.String(); got != tt.want {
				t.Errorf("PackageSystem.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBrewVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name: "valid version",
			input: `{
				"formulae": [{
					"versions": {
						"stable": "2.3.4"
					}
				}]
			}`,
			expect: "2.3.4",
		},
		{
			name:   "no formulae",
			input:  `{"formulae": []}`,
			expect: "",
		},
		{
			name:   "empty",
			input:  "",
			expect: "",
		},
		{
			name:   "invalid json",
			input:  `{invalid}`,
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBrewVersion(tt.input)
			if got != tt.expect {
				t.Errorf("parseBrewVersion() = %v, want %v", got, tt.expect)
			}
		})
	}
}
