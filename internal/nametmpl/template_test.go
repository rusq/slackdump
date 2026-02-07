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
package nametmpl

import (
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	type args struct {
		t string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"id is ok",
			args{"{{.ID}}"},
			mOK,
			false,
		},
		{
			"name is ok",
			args{"{{.Name}}"},
			mOK,
			false,
		},
		{
			"just threadTS is not ok",
			args{"{{.ThreadTS}}"},
			"",
			true,
		},
		{
			"threadTS and message ID is ok",
			args{"{{.ID}}-{{.ThreadTS}}"},
			"$$OK$$-$$PARTIAL$$",
			false,
		},
		{
			"threadTS and message ID is ok (conditional)",
			args{"{{.ID}}{{ if .ThreadTS}}-{{.ThreadTS}}{{end}}"},
			"$$OK$$-$$PARTIAL$$",
			false,
		},
		{
			"message is not ok",
			args{"{{.Message}}"},
			"",
			true,
		},
		{
			"unknown field is not ok",
			args{"{{.Who_dis}}"},
			"",
			true,
		},
		{
			"empty not ok",
			args{""},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compile(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			var buf strings.Builder
			if err := got.Execute(&buf, tc); err != nil {
				t.Errorf("Execute() error=%v", err)
			}
			if !strings.EqualFold(buf.String(), tt.want) {
				t.Errorf("rendered template mismatch:\nwant:\t%v\ngot:\n\t%v", tt.want, buf.String())
			}
		})
	}
}
