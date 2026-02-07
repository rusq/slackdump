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
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"go.uber.org/mock/gomock"
)

func init() {
	cfg.Log = slog.Default()
}

func Test_createWsp(t *testing.T) {
	type args struct {
		ctx       context.Context
		wsp       string
		confirmed bool
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*Mockmanager)
		wantErr  bool
	}{
		{
			name: "success", // I
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, nil)
				m.EXPECT().Select("test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "exist, ask- no", // VIII, II
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				canOverwrite = func(string) bool {
					// decline overwrite
					return false
				}
			},
			wantErr: true,
		},
		{
			name: "exist, skip interactive confirmation, but delete fails",
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: true,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				m.EXPECT().Delete("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "exist, ask- yes, delete fails", // VIII, III
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false, // so will ask
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(true)
				canOverwrite = func(string) bool {
					// confirm overwrite
					return true
				}
				m.EXPECT().Delete("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "auth fails", // IV, V
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, errors.New("fail"))
			},
			wantErr: true,
		},
		{
			name: "auth cancelled", // IV, IX
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, auth.ErrCancelled)
			},
			wantErr: true,
		},
		{
			name: "select fails", // I -> VII
			args: args{
				ctx:       t.Context(),
				wsp:       "test",
				confirmed: false,
			},
			expectFn: func(m *Mockmanager) {
				m.EXPECT().Exists("test").Return(false)
				m.EXPECT().Auth(gomock.Any(), "test", gomock.Any()).Return(nil, nil)
				m.EXPECT().Select("test").Return(errors.New("fail"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := NewMockmanager(ctrl)
			tt.expectFn(m)
			if err := createWsp(tt.args.ctx, m, tt.args.wsp, tt.args.confirmed); (err != nil) != tt.wantErr {
				t.Errorf("createWsp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_realname(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				name: "",
			},
			want: "default",
		},
		{
			name: "spaces",
			args: args{
				name: "  ",
			},
			want: "default",
		},
		{
			name: "test",
			args: args{
				name: "test",
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := realname(tt.args.name); got != tt.want {
				t.Errorf("realname() = %v, want %v", got, tt.want)
			}
		})
	}
}
