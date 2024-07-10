// Package pwcompat provides a compatibility layer, so when the playwright-go
// team decides to break compatibility again, there's a place to write a
// workaround.
package pwcompat

import "testing"

func Test_nvl(t *testing.T) {
	type args struct {
		first string
		rest  []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"first",
			args{"first", []string{"second", "third"}},
			"first",
		},
		{
			"second",
			args{"", []string{"second", "third"}},
			"second",
		},
		{
			"third",
			args{"", []string{"", "third"}},
			"third",
		},
		{
			"empty",
			args{"", []string{""}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nvl(tt.args.first, tt.args.rest...); got != tt.want {
				t.Errorf("nvl() = %v, want %v", got, tt.want)
			}
		})
	}
}
