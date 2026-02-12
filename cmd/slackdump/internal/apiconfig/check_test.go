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

package apiconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_runConfigCheck(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		content []byte
		wantErr bool
	}{
		{
			"arg set, file exists, contents valid",
			args{args: []string{filepath.Join(t.TempDir(), "test.yml")}},
			[]byte(sampleLimitsYaml),
			false,
		},
		{
			"arg not set",
			args{},
			nil,
			true,
		},
		{
			"arg set, file not exists",
			args{args: []string{"not_here$$$.$$$"}},
			nil,
			true,
		},
		{
			"arg set, file exists, contents invalid",
			args{args: []string{filepath.Join(t.TempDir(), "test1.yml")}},
			[]byte("workers:-500"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// if args and content is present, create this file.
			if len(tt.args.args) > 0 && len(tt.content) > 0 {
				if err := os.WriteFile(tt.args.args[0], tt.content, 0666); err != nil {
					t.Fatal(err)
				}
			}
			if err := runConfigCheck(t.Context(), CmdConfigCheck, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("runConfigCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
