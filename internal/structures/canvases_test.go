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

package structures_test

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/structures"
)

func TestHasCanvas(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		channel    *slack.Channel
		wantFileID string
		want       bool
	}{
		{
			name: "channel with canvas",
			channel: &slack.Channel{
				Properties: &slack.Properties{
					Canvas: slack.Canvas{IsEmpty: false, FileId: "file_id"},
				},
			},
			wantFileID: "file_id",
			want:       true,
		},
		{
			name: "channel without canvas",
			channel: &slack.Channel{
				Properties: &slack.Properties{
					Canvas: slack.Canvas{IsEmpty: true, FileId: ""},
				},
			},
			wantFileID: "",
			want:       false,
		},
		{
			name:       "channel with nil properties",
			channel:    &slack.Channel{Properties: nil},
			wantFileID: "",
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFileID, got := structures.CanvasFileID(tt.channel)
			if got != tt.want || gotFileID != tt.wantFileID {
				t.Errorf("CanvasFileID() = (%v, %v), want (%v, %v)", gotFileID, got, tt.wantFileID, tt.want)
			}
		})
	}
}

func TestCanvasChannelID(t *testing.T) {
	tests := []struct {
		name          string
		fileID        string
		wantChannelID string
		want          bool
	}{
		{
			name:          "valid canvas file ID",
			fileID:        "F1234567890",
			wantChannelID: "C1234567890",
			want:          true,
		},
		{
			name:   "empty file ID",
			fileID: "",
		},
		{
			name:   "non-canvas file ID",
			fileID: "D1234567890",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChannelID, got := structures.CanvasChannelID(tt.fileID)
			if got != tt.want || gotChannelID != tt.wantChannelID {
				t.Errorf("CanvasChannelID() = (%v, %v), want (%v, %v)", gotChannelID, got, tt.wantChannelID, tt.want)
			}
		})
	}
}
