package osext

import (
	"errors"
	"io/fs"
	"testing"
)

func TestIsPathError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil",
			args: args{err: nil},
			want: false,
		},
		{
			name: "fs.PathError",
			args: args{err: &fs.PathError{}},
			want: true,
		},
		{
			name: "error",
			args: args{err: errors.New("not a fs.PathError")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPathError(tt.args.err); got != tt.want {
				t.Errorf("IsPathError() = %v, want %v", got, tt.want)
			}
		})
	}
}
