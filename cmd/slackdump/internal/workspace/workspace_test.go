package workspace

import (
	"io"
	"strings"
	"testing"
)

func Test_currentWsp(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"ok",
			args{strings.NewReader("foo\n")},
			"foo",
		},
		{
			"empty",
			args{strings.NewReader("")},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readWsp(tt.args.r); got != tt.want {
				t.Errorf("currentWsp() = %v, want %v", got, tt.want)
			}
		})
	}
}
