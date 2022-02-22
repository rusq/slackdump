package slackdump

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseSlackTS(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"valid time", args{"1534552745.065949"}, time.Date(2018, 8, 18, 0, 39, 05, 65949, time.UTC), false},
		{"another valid time", args{"1638494510.037400"}, time.Date(2021, 12, 3, 1, 21, 50, 37400, time.UTC), false},
		{"time without millis", args{"0"}, time.Date(1970, 1, 1, 0, 00, 00, 0, time.UTC), false},
		{"invalid time", args{"x"}, time.Time{}, true},
		{"invalid time", args{"x.x"}, time.Time{}, true},
		{"invalid time", args{"4.x"}, time.Time{}, true},
		{"invalid time", args{"x.4"}, time.Time{}, true},
		{"invalid time", args{".4"}, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSlackTS(tt.args.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromSlackTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toSlackTime(t *testing.T) {
	type args struct {
		ts time.Time
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"ok", args{time.Date(2018, 8, 18, 0, 39, 05, 65949, time.UTC)}, "1534552745.065949"},
		{"another valid time", args{time.Date(2021, 12, 3, 1, 21, 50, 37400, time.UTC)}, "1638494510.037400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSlackTS(tt.args.ts); got != tt.want {
				t.Errorf("toSlackTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseThreadID(t *testing.T) {
	type args struct {
		threadID string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			"valid threadID",
			args{"p1577694990000400"},
			time.Date(2019, 12, 30, 8, 36, 30, 400, time.UTC),
			false,
		},
		{
			"empty",
			args{""},
			time.Time{},
			true,
		},
		{
			"corrupt threadID",
			args{"p1577694x90000400"},
			time.Time{},
			true,
		},
		{
			"invalid threadID",
			args{"1577694990000400"},
			time.Time{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseThreadID(tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseThreadID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseThreadID() = %v, want %v", got, tt.want)
			}
		})
	}
}
