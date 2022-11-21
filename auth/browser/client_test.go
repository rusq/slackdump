package browser

import (
	"reflect"
	"testing"
	"time"
)

func Test_extractToken(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"ok",
			args{"https://ora600.slack.com/api/api.features?_x_id=noversion-1651817410.129&token=xoxc-610187951300-604451271234-3473161557912-4c426dd426a45208707725b710302b32dda0ab002b80ccd8c4c8ac9971a11558&platform=sonic&_x_should_cache=false&_x_allow_cached=true&_x_team_id=THY5HTZ8U&_x_gantry=true&fp=7c\n"},
			"xoxc-610187951300-604451271234-3473161557912-4c426dd426a45208707725b710302b32dda0ab002b80ccd8c4c8ac9971a11558",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractToken(tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}

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
