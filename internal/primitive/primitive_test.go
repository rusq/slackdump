package primitive

import (
	"reflect"
	"testing"
)

func TestIfTrue(t *testing.T) {
	type args[T any] struct {
		cond bool
		t    T
		f    T
	}
	tests := []struct {
		name string
		args args[int]
		want int
	}{
		{
			"returns true",
			args[int]{true, 1, 0},
			1,
		},
		{
			"returns false",
			args[int]{false, 1, 0},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IfTrue(tt.args.cond, tt.args.t, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("iftrue() = %v, want %v", got, tt.want)
			}
		})
	}
}
