package auth

import (
	"errors"
	"fmt"
	"testing"
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
