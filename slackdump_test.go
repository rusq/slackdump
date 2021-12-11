package slackdump

import (
	"reflect"
	"testing"
	"time"
)

func Test_maxStringLength(t *testing.T) {
	type args struct {
		strings []string
	}
	tests := []struct {
		name       string
		args       args
		wantMaxlen int
	}{
		{"ascii", args{[]string{"123", "abc", "defg"}}, 4},
		{"unicode", args{[]string{"сообщение1", "проверка", "тест"}}, 10},
		{"empty", args{[]string{}}, 0},
		{"several empty", args{[]string{"", "", "", ""}}, 0},
		{"several empty one full", args{[]string{"", "", "1", ""}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotMaxlen := maxStringLength(tt.args.strings); gotMaxlen != tt.wantMaxlen {
				t.Errorf("maxStringLength() = %v, want %v", gotMaxlen, tt.wantMaxlen)
			}
		})
	}
}

func Test_fromSlackTime(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"good time", args{"1534552745.065949"}, time.Date(2018, 8, 18, 0, 39, 05, 65949, time.UTC), false},
		{"time without millis", args{"0"}, time.Date(1970, 1, 1, 0, 00, 00, 0, time.UTC), false},
		{"invalid time", args{"x"}, time.Time{}, true},
		{"invalid time", args{"x.x"}, time.Time{}, true},
		{"invalid time", args{"4.x"}, time.Time{}, true},
		{"invalid time", args{"x.4"}, time.Time{}, true},
		{"invalid time", args{".4"}, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fromSlackTime(tt.args.timestamp)
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
