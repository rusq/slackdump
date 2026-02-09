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

package repository

import (
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

var (
	file1 = &slack.File{
		ID:                 "FILE1",
		Created:            0,
		Timestamp:          slack.JSONTime(time.Date(1984, 9, 16, 15, 0, 0, 0, time.UTC).Unix()),
		Name:               "SOKO.COM",
		Title:              "Classic Sokoban Game, (c) 1984 Spectrum Holobyte",
		URLPrivateDownload: "https://archive.org/details/msdos_sokoban_1984_spectrum_holobyte",
		NumStars:           555,
		Mode:               "hosted",
	}

	dbFile1, _ = NewDBFile(1, 0, "C1", "1631820000.000000", "1531820000.000000", file1)
)

func TestNewDBFile(t *testing.T) {
	type args struct {
		chunkID     int64
		idx         int
		channelID   string
		threadTS    string
		parentMsgTS string
		file        *slack.File
	}
	tests := []struct {
		name    string
		args    args
		want    *DBFile
		wantErr bool
	}{
		{
			"success",
			args{1, 42, "C1", "1631820000.000000", "1531820000.000000", file1},
			&DBFile{
				ID:        "FILE1",
				ChunkID:   1,
				ChannelID: "C1",
				MessageID: ptr[int64](1531820000000000),
				ThreadID:  ptr[int64](1631820000000000),
				Index:     42,
				Mode:      "hosted",
				Filename:  ptr("SOKO.COM"),
				URL:       ptr("https://archive.org/details/msdos_sokoban_1984_spectrum_holobyte"),
				Data:      must(marshal(file1)),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBFile(tt.args.chunkID, tt.args.idx, tt.args.channelID, tt.args.threadTS, tt.args.parentMsgTS, tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
