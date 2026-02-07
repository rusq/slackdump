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
package archive

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_stopFn_Stop(t *testing.T) {
	noError := func() error { return nil }
	yesError := func() error { return assert.AnError }
	tests := []struct {
		name    string
		s       stopFn
		wantErr bool
	}{
		{
			name: "no error",
			s: stopFn{
				noError,
				noError,
				noError,
			},
			wantErr: false,
		},
		{
			name: "yes error",
			s: stopFn{
				noError,
				noError,
				yesError,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("stopFn.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
