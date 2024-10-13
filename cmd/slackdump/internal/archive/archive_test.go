package archive

import (
	_ "embed"
	"testing"
)

func TestStripZipExt(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"zip",
			args{"foo.zip"},
			"foo",
		},
		{
			"tar.gz",
			args{"foo.tar.gz"},
			"foo.tar.gz",
		},
		{
			"ZIP",
			args{"foo.ZIP"},
			"foo",
		},
		{
			"empty",
			args{""},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripZipExt(tt.args.s); got != tt.want {
				t.Errorf("StripZipExt() = %v, want %v", got, tt.want)
			}
		})
	}
}
