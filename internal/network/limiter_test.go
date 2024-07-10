package network

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestNewLimiter(t *testing.T) {
	type args struct {
		t     Tier
		burst uint
		boost int
	}
	tests := []struct {
		name       string
		args       args
		want       *rate.Limiter
		wantPerSec rate.Limit
	}{
		{
			name: "tier 2",
			args: args{
				t:     Tier2,
				burst: 10,
				boost: 0,
			},
			wantPerSec: 0.3333333333333333,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLimiter(tt.args.t, tt.args.burst, tt.args.boost); got.Limit() != tt.wantPerSec {
				t.Errorf("NewLimiter() = %v, want %v", got.Limit(), tt.wantPerSec)
			}
		})
	}
}

func Test_every(t *testing.T) {
	type args struct {
		t     Tier
		boost int
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "tier 2",
			args: args{
				t:     Tier2,
				boost: 0,
			},
			want: time.Minute / 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := every(tt.args.t, tt.args.boost); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("every() = %s, want %s", got, tt.want)
			}
		})
	}
}
