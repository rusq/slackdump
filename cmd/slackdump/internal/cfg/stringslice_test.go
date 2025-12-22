package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSlice_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		ss   *StringSlice
		args args
		want StringSlice
	}{
		{
			name: "sets the string slice",
			ss:   new(StringSlice),
			args: args{"alpha,bravo,charlie"},
			want: StringSlice{"alpha", "bravo", "charlie"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ss.Set(tt.args.s)
			assert.Equal(t, tt.want, *tt.ss)
		})
	}
}

func TestStringSlice_String(t *testing.T) {
	tests := []struct {
		name string
		ss   *StringSlice
		want string
	}{
		{
			name: "abc",
			ss:   &StringSlice{"alpha", "bravo", "charlie"},
			want: "alpha,bravo,charlie",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ss.String(); got != tt.want {
				t.Errorf("StringSlice.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
