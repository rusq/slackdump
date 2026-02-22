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
	"fmt"
	"strings"
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

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		v1      string
		v2      string
		want    int
		wantErr bool
	}{
		{
			name: "2.10.0 > 2.9.0",
			v1:   "2.10.0",
			v2:   "2.9.0",
			want: 1,
		},
		{
			name: "2.9.0 < 2.10.0",
			v1:   "2.9.0",
			v2:   "2.10.0",
			want: -1,
		},
		{
			name: "2.10.0 == 2.10.0",
			v1:   "2.10.0",
			v2:   "2.10.0",
			want: 0,
		},
		{
			name: "3.0.0 > 2.10.0",
			v1:   "3.0.0",
			v2:   "2.10.0",
			want: 1,
		},
		{
			name: "2.10.1 > 2.10.0",
			v1:   "2.10.1",
			v2:   "2.10.0",
			want: 1,
		},
		{
			name: "2.10.0 > 2.9.99",
			v1:   "2.10.0",
			v2:   "2.9.99",
			want: 1,
		},
		{
			name: "1.0.0 < 1.0.1",
			v1:   "1.0.0",
			v2:   "1.0.1",
			want: -1,
		},
		{
			name: "different lengths: 2.10 == 2.10.0",
			v1:   "2.10",
			v2:   "2.10.0",
			want: 0,
		},
		{
			name: "different lengths: 2.10.1 > 2.10",
			v1:   "2.10.1",
			v2:   "2.10",
			want: 1,
		},
		{
			name:    "invalid version v1",
			v1:      "2.a.0",
			v2:      "2.10.0",
			wantErr: true,
		},
		{
			name:    "invalid version v2",
			v1:      "2.10.0",
			v2:      "2.b.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareVersions(tt.v1, tt.v2)
			if (err != nil) != tt.wantErr {
				t.Errorf("compareVersions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("compareVersions(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsBrewOutdated(t *testing.T) {
	tests := []struct {
		name          string
		brewVersion   string
		latestVersion string
		wantOutdated  bool
		wantBrewVer   string
	}{
		{
			name:          "brew outdated: 2.9.0 < 2.10.0",
			brewVersion:   "2.9.0",
			latestVersion: "2.10.0",
			wantOutdated:  true,
			wantBrewVer:   "2.9.0",
		},
		{
			name:          "brew up-to-date: 2.10.0 == 2.10.0",
			brewVersion:   "2.10.0",
			latestVersion: "2.10.0",
			wantOutdated:  false,
			wantBrewVer:   "2.10.0",
		},
		{
			name:          "brew ahead: 2.11.0 > 2.10.0",
			brewVersion:   "2.11.0",
			latestVersion: "2.10.0",
			wantOutdated:  false,
			wantBrewVer:   "2.11.0",
		},
		{
			name:          "with v prefix",
			brewVersion:   "v2.9.0",
			latestVersion: "v2.10.0",
			wantOutdated:  true,
			wantBrewVer:   "2.9.0",
		},
		{
			name:          "major version difference",
			brewVersion:   "1.9.0",
			latestVersion: "2.0.0",
			wantOutdated:  true,
			wantBrewVer:   "1.9.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock brew info JSON
			brewJSON := fmt.Sprintf(`{
				"formulae": [{
					"versions": {
						"stable": "%s"
					}
				}]
			}`, tt.brewVersion)

			// We can't easily mock exec.CommandContext, so we'll test compareVersions separately
			// and just verify the logic here
			v1 := strings.TrimPrefix(tt.brewVersion, "v")
			v2 := strings.TrimPrefix(tt.latestVersion, "v")

			cmp, err := compareVersions(v1, v2)
			if err != nil {
				t.Fatalf("compareVersions() error = %v", err)
			}

			gotOutdated := cmp < 0
			if gotOutdated != tt.wantOutdated {
				t.Errorf("IsBrewOutdated logic: outdated = %v, want %v (comparing %s vs %s, cmp=%d)",
					gotOutdated, tt.wantOutdated, v1, v2, cmp)
			}

			// Verify brew JSON parsing works
			gotVer := parseBrewVersion(brewJSON)
			if gotVer != tt.brewVersion {
				t.Errorf("parseBrewVersion() = %v, want %v", gotVer, tt.brewVersion)
			}
		})
	}
}
