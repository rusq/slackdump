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
package auth

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rusq/slack"
)

var errSample = errors.New("test error")

func TestAuthError_Unwrap(t *testing.T) {
	type fields struct {
		Err error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			"unwrap unwraps properly",
			fields{Err: errSample},
			errSample,
		},
		{
			"multilevel wrap",
			fields{Err: fmt.Errorf("blah: %w", errSample)},
			errSample,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &Error{
				Err: tt.fields.Err,
			}
			if err := ae.Unwrap(); (err != nil) && !errors.Is(err, tt.wantErr) {
				t.Errorf("AuthError.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthError_Is(t *testing.T) {
	type fields struct {
		Err error
	}
	type args struct {
		target error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			"is correctly compares underlying error",
			fields{Err: errSample},
			args{errSample},
			true,
		},
		{
			"not matching error returns false",
			fields{Err: errors.New("not me bro")},
			args{errSample},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &Error{
				Err: tt.fields.Err,
			}
			if got := ae.Is(tt.args.target); got != tt.want {
				t.Errorf("AuthError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidAuthErr(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"not an auth error",
			args{
				errors.New("not me bro"),
			},
			false,
		},
		{
			"auth error",
			args{
				&Error{
					Err: slack.SlackErrorResponse{
						Err: "invalid_auth",
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidAuthErr(tt.args.err); got != tt.want {
				t.Errorf("IsInvalidAuthErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
