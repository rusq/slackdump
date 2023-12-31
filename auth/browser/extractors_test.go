package browser

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

const testMultipart = "-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"token\"\r\n\r\nxoxc-888888888888-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"platform\"\r\n\r\nsonic\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"_x_should_cache\"\r\n\r\nfalse\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"_x_allow_cached\"\r\n\r\ntrue\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"_x_team_id\"\r\n\r\nTFCSDNRL5\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"_x_gantry\"\r\n\r\ntrue\r\n-----------------------------37168696061856579082739228613\r\nContent-Disposition: form-data; name=\"_x_sonic\"\r\n\r\ntrue\r\n-----------------------------37168696061856579082739228613--\r\n"

var testHdrValues = "multipart/form-data; boundary=---------------------------37168696061856579082739228613"

const testBoundary = "---------------------------37168696061856579082739228613"

func Test_extractTokenGet(t *testing.T) {
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
			got, err := extractTokenGet(tt.args.uri)
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

func Test_tokenFromMultipart(t *testing.T) {
	type args struct {
		s        string
		boundary string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"ok", args{testMultipart, testBoundary}, "xoxc-888888888888-888888888888-8888888888888-fffffffffffffffa915fe069d70a8ad81743b0ec4ee9c81540af43f5e143264b", false},
		{"bad boundary", args{testMultipart, "bad"}, "", true},
		{"bad multipart", args{"bad", testBoundary}, "", true},
		{"empty", args{"", testBoundary}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenFromMultipart(tt.args.s, tt.args.boundary)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractTokenPost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractTokenPost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_boundary(t *testing.T) {
	tests := []struct {
		name    string
		expect  func(r *MockRequest)
		want    string
		wantErr bool
	}{
		{
			"ok",
			func(r *MockRequest) {
				r.EXPECT().HeaderValue("Content-Type").Return(testHdrValues, nil)
			},
			testBoundary,
			false,
		},
		{
			"no header",
			func(r *MockRequest) {
				r.EXPECT().HeaderValue("Content-Type").Return("", nil)
			},
			"",
			true,
		},
		{
			"bad header",
			func(r *MockRequest) {
				r.EXPECT().HeaderValue("Content-Type").Return("bad", nil)
			},
			"",
			true,
		},
		{
			"error",
			func(r *MockRequest) {
				r.EXPECT().HeaderValue("Content-Type").Return("", errors.New("bad"))
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mr := NewMockRequest(ctrl)
			tt.expect(mr)
			got, err := boundary(mr)
			if (err != nil) != tt.wantErr {
				t.Errorf("boundary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("boundary() = %v, want %v", got, tt.want)
			}
		})
	}
}
