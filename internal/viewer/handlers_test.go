package viewer

import "testing"

func Test_isInvalid(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"relative path", args{"../test.txt"}, true},
		{"home dir ref", args{"~/test.txt"}, true},
		{"filename with tilda #561", args{"test~1.txt"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInvalid(tt.args.path); got != tt.want {
				t.Errorf("isInvalid() = %v, want %v", got, tt.want)
			}
		})
	}
}
