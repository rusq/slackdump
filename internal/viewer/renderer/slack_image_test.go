package renderer

import (
	"testing"

	"github.com/rusq/slack"
)

func TestSlack_mbtImage(t *testing.T) {
	type fields struct {
		uu map[string]slack.User
		cc map[string]slack.Channel
	}
	type args struct {
		ib slack.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Slack{
				uu: tt.fields.uu,
				cc: tt.fields.cc,
			}
			got, got1, err := s.mbtImage(tt.args.ib)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slack.mbtImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Slack.mbtImage() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Slack.mbtImage() = %v, want %v", got1, tt.want1)
			}
		})
	}
}
