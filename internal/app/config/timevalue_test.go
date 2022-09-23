package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func tv(t time.Time) *TimeValue {
	tv := TimeValue(t)
	return &tv
}

func TestTimeValue_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name     string
		tv       *TimeValue
		args     args
		wantTime *TimeValue
		wantErr  bool
	}{
		{
			"valid value",
			&TimeValue{},
			args{"2009-09-16T20:30:40"},
			tv(time.Date(2009, 9, 16, 20, 30, 40, 0, time.UTC)),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := &TimeValue{}
			if err := tv.Set(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("TimeValue.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantTime, tv)
		})
	}
}
