package chunk

import "testing"

func Test_hash(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{""},
			"da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			"hello",
			args{"hello"},
			"aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hash(tt.args.s); got != tt.want {
				t.Errorf("hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_hash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		hash("hello")
	}
}
