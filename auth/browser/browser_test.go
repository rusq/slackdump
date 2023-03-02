package browser

import (
	"reflect"
	"testing"
	"time"
)

func Test_float2time(t *testing.T) {
	type args struct {
		v float64
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{"ok", args{1.68335956e+09}, time.Unix(1683359560, 0)},
		{"stripped", args{1.6544155598311e+09}, time.Unix(1654415559, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := float2time(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("float2time() = %v, want %v", got, tt.want)
			}
		})
	}
}
