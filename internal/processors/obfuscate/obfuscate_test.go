package obfuscate

import (
	"math/rand"
	"testing"
)

func init() {
	rand.Seed(0) // make it deterministic
}

func Test_randomString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{n: 0},
			want: "jXUJR9JT5pul5g8MDbK7E1ycTwBhzdJG9 ",
		},
		{
			name: "one",
			args: args{n: 1},
			want: "VwGabEN7FkWNmyD0HtOdvcYYvfHfF hVA6",
		},
		{
			name: "100",
			args: args{n: 100},
			want: "d1BtVOw52BH40tQ4xsZr1rbOEdndtLrooKH5L9GzLgWmmWfVTBKfSvym98qEQMYaWdLEKrJCEXzYB2bFiOLzhKfgf0hdxneHm6GIP4BlU7M3cWoFQL4mevBBbRf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := randomString(tt.args.n); got != tt.want {
				t.Errorf("randomString() = %v, want %v", got, tt.want)
			}
		})
	}
}
