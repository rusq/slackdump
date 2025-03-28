package pool

import (
	"testing"
)

func Test_roundRobin_next(t *testing.T) {
	type fields struct {
		total int
		i     int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "test1",
			fields: fields{total: 3, i: 0},
			want:   1,
		},
		{
			name:   "test2",
			fields: fields{total: 3, i: 1},
			want:   2,
		},
		{
			name:   "test3",
			fields: fields{total: 3, i: 2},
			want:   0,
		},
		{
			name:   "test4",
			fields: fields{total: 2, i: 0},
			want:   1,
		},
		{
			name:   "test5",
			fields: fields{total: 2, i: 1},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &roundRobin{
				total: tt.fields.total,
				i:     tt.fields.i,
			}
			if got := r.next(); got != tt.want {
				t.Errorf("roundRobin.next() = %v, want %v", got, tt.want)
			}
		})
	}
}
