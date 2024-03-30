package primitive

import "testing"

func Test_counter_Add(t *testing.T) {
	type fields struct {
		n int
	}
	type args struct {
		n int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "add",
			fields: fields{
				n: 0,
			},
			args: args{
				n: 1,
			},
			want: 1,
		},
		{
			name: "add n",
			fields: fields{
				n: 10,
			},
			args: args{
				n: 20,
			},
			want: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Counter{
				n: tt.fields.n,
			}
			if got := ep.Add(tt.args.n); got != tt.want {
				t.Errorf("counter.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_counter_Inc(t *testing.T) {
	type fields struct {
		n int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "add to initial 0",
			fields: fields{},
			want:   1,
		},
		{
			name: "increment",
			fields: fields{
				n: 10,
			},
			want: 11,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Counter{
				n: tt.fields.n,
			}
			if got := ep.Inc(); got != tt.want {
				t.Errorf("counter.Inc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_counter_Dec(t *testing.T) {
	type fields struct {
		n int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "dec initial 0",
			fields: fields{},
			want:   -1,
		},
		{
			name: "decrement",
			fields: fields{
				n: 10,
			},
			want: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Counter{
				n: tt.fields.n,
			}
			if got := ep.Dec(); got != tt.want {
				t.Errorf("counter.Dec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_counter_N(t *testing.T) {
	type fields struct {
		n int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "initial 0",
			fields: fields{},
			want:   0,
		},
		{
			name: "10",
			fields: fields{
				n: 10,
			},
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Counter{
				n: tt.fields.n,
			}
			if got := c.N(); got != tt.want {
				t.Errorf("counter.N() = %v, want %v", got, tt.want)
			}
		})
	}
}
