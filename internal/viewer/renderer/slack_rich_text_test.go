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
		{
			"multiline",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New\nMessage", nil)),
			},
			"New<br>Message",
			false,
		},
		{
			"bold",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Bold: true})),
			},
			"<b>New Message</b>",
			false,
		},
		{
			"italic",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Italic: true})),
			},
			"<i>New Message</i>",
			false,
		},
		{
			"strike",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Strike: true})),
			},
			"<s>New Message</s>",
			false,
		},
		{
			"code",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("Code message", &slack.RichTextSectionTextStyle{Code: true})),
			},
			"<code>Code message</code>",
			false,
		},
		{
			"bold italic",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Bold: true, Italic: true})),
			},
			"<i><b>New Message</b></i>",
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

func Test_rtseLink(t *testing.T) {
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
			"valid link",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionLinkElement("https://example.com", "example.com", nil)),
			},
			"<a href=\"https://example.com\">example.com</a>",
			false,
		},
		{
			"empty text",
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionLinkElement("https://example.com", "", nil)),
			},
			"<a href=\"https://example.com\">https://example.com</a>",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rtseLink(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("rtseLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rtseLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
