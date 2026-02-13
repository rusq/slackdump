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

package structures

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseSlackTS(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"valid time", args{"1534552745.065949"}, time.UnixMicro(1534552745065949).UTC(), false},
		{"another valid time", args{"1638494510.037400"}, time.Date(2021, 12, 3, 1, 21, 50, 37400000, time.UTC), false},
		{"the time when I slipped", args{"1645551829.244659"}, time.Date(2022, 2, 22, 17, 43, 49, 244659000, time.UTC), false},
		{"time without millis", args{"0"}, time.Date(1970, 1, 1, 0, 0o0, 0o0, 0, time.UTC), false},
		{"invalid time", args{"x"}, time.Time{}, true},
		{"invalid time", args{"x.x"}, time.Time{}, true},
		{"invalid time", args{"4.x"}, time.Time{}, true},
		{"invalid time", args{"x.4"}, time.Time{}, true},
		{"invalid time", args{".4"}, time.Time{}, true},
		{"polly time", args{"1737160363.583369"}, time.UnixMicro(1737160363583369).UTC(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSlackTS(tt.args.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromSlackTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_FormatSlackTS(t *testing.T) {
	type args struct {
		ts time.Time
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"ok", args{time.Date(2018, 8, 18, 0, 39, 0o5, 65949000, time.UTC)}, "1534552745.065949"},
		{"another valid time", args{time.Date(2021, 12, 3, 1, 21, 50, 37400000, time.UTC)}, "1638494510.037400"},
		{"empty", args{time.Time{}}, ""},
		{"Happy new 1970 year", args{time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)}, "0.000000"},
		{"prepare for the future", args{time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(-1 * time.Nanosecond)}, ""},
		{"polly message", args{time.UnixMicro(1737160363583369)}, "1737160363.583369"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatSlackTS(tt.args.ts); got != tt.want {
				t.Errorf("toSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseThreadID(t *testing.T) {
	type args struct {
		threadID string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			"valid threadID",
			args{"p1577694990000400"},
			time.Date(2019, 12, 30, 8, 36, 30, 400000, time.UTC),
			false,
		},
		{
			"empty",
			args{""},
			time.Time{},
			true,
		},
		{
			"corrupt threadID",
			args{"p1577694x90000400"},
			time.Time{},
			true,
		},
		{
			"invalid threadID",
			args{"1577694990000400"},
			time.Time{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseThreadID(tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseThreadID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseThreadID() = %v, want %v", got, tt.want)
			}
		})
	}
}
