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
package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/rusq/slackdump/v4/internal/cache"
	gomock "go.uber.org/mock/gomock"
)

func Test_printBare(t *testing.T) {
	type args struct {
		_          manager
		current    string
		workspaces []string
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"Test 1",
			args{
				current:    "current",
				workspaces: []string{"workspace1", "workspace2", "current"},
			},
			"workspace1\nworkspace2\n*current\n",
			false,
		},
		{
			"Test 2",
			args{
				current:    "workspace1",
				workspaces: []string{"workspace1", "workspace2", "notcurrent"},
			},
			"*workspace1\nworkspace2\nnotcurrent\n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := printBare(t.Context(), w, nil, tt.args.current, tt.args.workspaces); (err != nil) != tt.wantErr {
				t.Errorf("printBare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("printBare() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func Test_list(t *testing.T) {
	type args struct {
		// m         manager
		formatter formatFunc
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*Mockmanager) error
		wantErr  bool
	}{
		{
			"happy path",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return([]string{"workspace1", "workspace2"}, nil)
				mm.EXPECT().Current().Return("workspace1", nil)
				return nil
			},
			false,
		},
		{
			"error getting list",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return(nil, errors.New("error getting list"))
				return nil
			},
			true,
		},
		{
			"error getting list, no workspaces",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return(nil, cache.ErrNoWorkspaces)
				return nil
			},
			true,
		},
		{
			"error getting current",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return([]string{"workspace1", "workspace2"}, nil)
				mm.EXPECT().Current().Return("", errors.New("error getting current"))
				return nil
			},
			true,
		},
		{
			"error getting current, no default, select error",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return([]string{"workspace1", "workspace2"}, nil)
				mm.EXPECT().Current().Return("", cache.ErrNoDefault)
				mm.EXPECT().Select("workspace1").Return(errors.New("error selecting workspace"))
				return nil
			},
			true,
		},
		{
			"error getting current, no default, select ok",
			args{formatter: testFmt},
			func(mm *Mockmanager) error {
				mm.EXPECT().List().Return([]string{"workspace1", "workspace2"}, nil)
				mm.EXPECT().Current().Return("", cache.ErrNoDefault)
				mm.EXPECT().Select("workspace1").Return(nil)
				return nil
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mm := NewMockmanager(ctrl)
			tt.expectFn(mm)
			if err := list(t.Context(), mm, tt.args.formatter); (err != nil) != tt.wantErr {
				t.Errorf("list() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func testFmt(_ context.Context, w io.Writer, m manager, current string, wsps []string) error {
	for _, wsp := range wsps {
		if wsp == current {
			fmt.Fprint(w, ">")
		}
		fmt.Fprintf(w, "%s\n", wsp)
	}
	return nil
}
