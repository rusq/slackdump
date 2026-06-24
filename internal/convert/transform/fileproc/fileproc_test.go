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

// Package fileproc is the file processor that can be used in conjunction with
// the transformer.  It downloads files to the local filesystem using the
// provided downloader.  Probably it's a good idea to use the
// [downloader.Client] for this.
package fileproc

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/fixtures"
)

func TestIsValid(t *testing.T) {
	type args struct {
		f *slack.File
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"valid file",
			args{fixtures.LoadPtr[slack.File](fixtures.FileJPEG)},
			true,
		},
		{
			"tombstone",
			args{&slack.File{Mode: "tombstone", Name: "foo"}},
			false,
		},
		{
			"external file",
			args{&slack.File{Mode: "external", Name: "foo", IsExternal: true}},
			false,
		},
		{
			"hidden by limit",
			args{&slack.File{Mode: "hidden_by_limit", Name: "foo"}},
			false,
		},
		{
			"tombstone",
			args{&slack.File{Mode: "tombstone", Name: "foo"}},
			false,
		},
		{
			"external file",
			args{&slack.File{Mode: "", Name: "foo", IsExternal: true}},
			true,
		},
		{
			"external false name is not empty",
			args{&slack.File{Mode: "", Name: "foo", IsExternal: false}},
			true,
		},
		{
			"empty name",
			args{&slack.File{Mode: "", Name: "", IsExternal: false}},
			false,
		},
		{
			"nil file",
			args{nil},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.args.f); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkipReason(t *testing.T) {
	tests := []struct {
		name       string
		file       *slack.File
		wantReason string
		wantSkip   bool
	}{
		{
			name:       "tombstone",
			file:       &slack.File{Mode: "tombstone", Name: "foo"},
			wantReason: "tombstone",
			wantSkip:   true,
		},
		{
			name:       "hidden by limit",
			file:       &slack.File{Mode: "hidden_by_limit", Name: "foo"},
			wantReason: "hidden_by_limit",
			wantSkip:   true,
		},
		{
			name:       "external",
			file:       &slack.File{Mode: "external", Name: "foo", IsExternal: true},
			wantReason: "external",
			wantSkip:   true,
		},
		{
			name:     "downloadable file",
			file:     fixtures.LoadPtr[slack.File](fixtures.FileJPEG),
			wantSkip: false,
		},
		{
			name:     "nil file",
			file:     nil,
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReason, gotSkip := SkipReason(tt.file)
			if gotReason != tt.wantReason || gotSkip != tt.wantSkip {
				t.Fatalf("SkipReason() = (%q, %v), want (%q, %v)", gotReason, gotSkip, tt.wantReason, tt.wantSkip)
			}
			if got := ShouldSkip(tt.file); got != tt.wantSkip {
				t.Fatalf("ShouldSkip() = %v, want %v", got, tt.wantSkip)
			}
		})
	}
}

func TestIsValidWithReason(t *testing.T) {
	tests := []struct {
		name    string
		file    *slack.File
		wantErr bool
	}{
		{
			name:    "tombstone is skipped not invalid",
			file:    &slack.File{Mode: "tombstone", Name: "foo"},
			wantErr: false,
		},
		{
			name:    "hidden by limit is skipped not invalid",
			file:    &slack.File{Mode: "hidden_by_limit", Name: "foo"},
			wantErr: false,
		},
		{
			name:    "external is skipped not invalid",
			file:    &slack.File{Mode: "external", Name: "foo", IsExternal: true},
			wantErr: false,
		},
		{
			name:    "nil file is invalid",
			file:    nil,
			wantErr: true,
		},
		{
			name:    "downloadable file with empty name is invalid",
			file:    &slack.File{ID: "F123", Name: "", IsExternal: false},
			wantErr: true,
		},
		{
			name:    "valid downloadable file",
			file:    fixtures.LoadPtr[slack.File](fixtures.FileJPEG),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidWithReason(tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("IsValidWithReason() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
