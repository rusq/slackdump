package auth_ui

import "testing"

func TestSanitize(t *testing.T) {
	type args struct {
		workspace string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"not a URL", args{"blahblah"}, "blahblah", false},
		{"url slash", args{"https://blahblah.slack.com/"}, "blahblah", false},
		{"url no slash", args{"https://blahblah.slack.com"}, "blahblah", false},
		{"url no schema slash", args{"blahblah.slack.com/"}, "blahblah", false},
		{"url no schema no slash", args{"blahblah.slack.com"}, "blahblah", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Sanitize(tt.args.workspace)
			if (err != nil) != tt.wantErr {
				t.Errorf("sanitize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("sanitize() got = %v, want %v", got, tt.want)
			}
		})
	}
}
