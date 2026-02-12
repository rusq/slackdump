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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func tv(t time.Time) *TimeValue {
	tv := TimeValue(t)
	return &tv
}

func TestTimeValue_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name     string
		tv       *TimeValue
		args     args
		wantTime *TimeValue
		wantErr  bool
	}{
		{
			"valid value",
			&TimeValue{},
			args{"2009-09-16T20:30:40"},
			tv(time.Date(2009, 9, 16, 20, 30, 40, 0, time.UTC)),
			false,
		},
		{
			"empty value",
			&TimeValue{},
			args{""},
			tv(time.Time{}),
			false,
		},
		{
			"invalid value",
			&TimeValue{},
			args{"invalid"},
			tv(time.Time{}),
			true,
		},
		{
			"date value",
			&TimeValue{},
			args{"2009-09-16"},
			tv(time.Date(2009, 9, 16, 0, 0, 0, 0, time.UTC)),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := &TimeValue{}
			if err := tv.Set(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("TimeValue.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantTime, tv)
		})
	}
}

func TestTimeValue_String(t *testing.T) {
	tests := []struct {
		name string
		tv   *TimeValue
		want string
	}{
		{
			"zero value",
			tv(time.Time{}),
			"",
		},
		{
			"valid value",
			tv(time.Date(2009, 9, 16, 20, 30, 40, 0, time.UTC)),
			"2009-09-16T20:30:40",
		},
		{
			"round date",
			tv(time.Date(2009, 9, 16, 0, 0, 0, 0, time.UTC)),
			"2009-09-16",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := tt.tv
			if got := tv.String(); got != tt.want {
				t.Errorf("TimeValue.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
