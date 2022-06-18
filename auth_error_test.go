package slackdump

import (
	"errors"
	"fmt"
	"testing"
)

var testErr = errors.New("test error")

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
			fields{Err: testErr},
			testErr,
		},
		{
			"multilevel wrap",
			fields{Err: fmt.Errorf("blah: %w", testErr)},
			testErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &AuthError{
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
			fields{Err: testErr},
			args{testErr},
			true,
		},
		{
			"not matching error returns false",
			fields{Err: errors.New("not me bro")},
			args{testErr},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ae := &AuthError{
				Err: tt.fields.Err,
			}
			if got := ae.Is(tt.args.target); got != tt.want {
				t.Errorf("AuthError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}
