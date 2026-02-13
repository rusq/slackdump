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

package renderer

import (
	"testing"

	"github.com/rusq/slack"
)

func TestSlack_mbtImage(t *testing.T) {
	type fields struct {
		uu map[string]slack.User
		cc map[string]slack.Channel
	}
	type args struct {
		ib slack.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slack{
				uu: tt.fields.uu,
				cc: tt.fields.cc,
			}
			got, got1, err := s.mbtImage(tt.args.ib)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slack.mbtImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Slack.mbtImage() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Slack.mbtImage() = %v, want %v", got1, tt.want1)
			}
		})
	}
}
