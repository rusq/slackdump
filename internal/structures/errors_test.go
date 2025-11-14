package structures

import (
	"io"
	"testing"

	"github.com/rusq/slack"
)

func TestIsSlackResponseError(t *testing.T) {
	type args struct {
		e error
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"test error",
			args{
				slack.SlackErrorResponse{
					Err: "test error",
				},
				"test error",
			},
			true,
		},
		{
			"different error text",
			args{
				slack.SlackErrorResponse{
					Err: "another error",
				},
				"test error",
			},
			false,
		},
		{
			"different error",
			args{
				io.EOF,
				"test error",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSlackResponseError(tt.args.e, tt.args.s); got != tt.want {
				t.Errorf("IsSlackResponseError() = %v, want %v", got, tt.want)
			}
		})
	}
}
