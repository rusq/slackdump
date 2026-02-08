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
package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimits_Apply(t *testing.T) {
	type fields struct {
		Workers         int
		DownloadRetries int
		Tier2           TierLimit
		Tier3           TierLimit
		Tier4           TierLimit
		Request         RequestLimit
	}
	type args struct {
		other Limits
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Limits
		wantErr bool
	}{
		{
			"boost",
			fields(DefLimits),
			args{
				other: Limits{
					Workers:         DefLimits.Workers,
					DownloadRetries: DefLimits.DownloadRetries,
					Tier2:           TierLimit{Burst: DefLimits.Tier2.Burst, Boost: 0},
					Tier3:           TierLimit{Burst: DefLimits.Tier2.Burst, Boost: 0},
					Tier4:           TierLimit{Burst: DefLimits.Tier4.Burst, Boost: 0},
					Request:         DefLimits.Request,
				},
			},
			Limits{
				Workers:         DefLimits.Workers,
				DownloadRetries: DefLimits.DownloadRetries,
				Tier2:           TierLimit{Burst: 1, Boost: 0},
				Tier3:           TierLimit{Burst: 1, Boost: 0},
				Request:         DefLimits.Request,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Limits{
				Workers:         tt.fields.Workers,
				DownloadRetries: tt.fields.DownloadRetries,
				Tier2:           tt.fields.Tier2,
				Tier3:           tt.fields.Tier3,
				Request:         tt.fields.Request,
			}
			if err := o.Apply(tt.args.other); (err != nil) != tt.wantErr {
				t.Errorf("o.Apply() error=%v wantErr=%v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestLimits_Validate(t *testing.T) {
	type fields struct {
		Workers         int
		DownloadRetries int
		Tier2           TierLimit
		Tier3           TierLimit
		Tier4           TierLimit
		Request         RequestLimit
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{"validate default options",
			fields(DefLimits),
			func(t assert.TestingT, err error, i ...interface{}) bool {
				return err == nil
			},
		},
		{
			"empty options is an error",
			fields{},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				if err == nil {
					t.Errorf("expected error, but got %v", err)
					return false
				}
				return true
			},
		},
		{
			"invalid workers",
			fields{
				Workers: -1,
				Tier2:   TierLimit{Burst: 1},
				Tier3:   TierLimit{Burst: 1},
				Request: RequestLimit{
					Conversations: 50,
					Replies:       50,
					Channels:      50,
				},
			},
			func(t assert.TestingT, err error, i ...interface{}) bool {
				if err == nil {
					t.Errorf("expected error, but got %v", err)
				}
				return err != nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Limits{
				Workers:         tt.fields.Workers,
				DownloadRetries: tt.fields.DownloadRetries,
				Tier2:           tt.fields.Tier2,
				Tier3:           tt.fields.Tier3,
				Request:         tt.fields.Request,
			}
			tt.wantErr(t, o.Validate(), "Validate()")
		})
	}
}
