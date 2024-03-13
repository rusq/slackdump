package renderer

import (
	"testing"

	"github.com/rusq/slack"
)

func Test_rtseText(t *testing.T) {
	type args struct {
		ie slack.RichTextSectionElement
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"valid text section",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", nil)),
			},
			"New Message",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rtseText(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("rtseText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rtseText() = %v, want %v", got, tt.want)
			}
		})
	}
}
