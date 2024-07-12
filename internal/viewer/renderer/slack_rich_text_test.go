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
		s       *Slack
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"valid text section",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", nil)),
			},
			"New Message",
			"",
			false,
		},
		{
			"multiline",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New\nMessage", nil)),
			},
			"New<br>Message",
			"",
			false,
		},
		{
			"bold",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Bold: true})),
			},
			"<b>New Message</b>",
			"",
			false,
		},
		{
			"italic",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Italic: true})),
			},
			"<i>New Message</i>",
			"",
			false,
		},
		{
			"strike",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Strike: true})),
			},
			"<s>New Message</s>",
			"",
			false,
		},
		{
			"code",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("Code message", &slack.RichTextSectionTextStyle{Code: true})),
			},
			"<code>Code message</code>",
			"",
			false,
		},
		{
			"bold italic",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionTextElement("New Message", &slack.RichTextSectionTextStyle{Bold: true, Italic: true})),
			},
			"<i><b>New Message</b></i>",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.s.rtseText(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("rtseText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rtseText() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("rtseText() = %v, want1 %v", got1, tt.want1)
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
		s       *Slack
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"valid link",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionLinkElement("https://example.com", "example.com", nil)),
			},
			"<a href=\"https://example.com\">example.com</a>",
			"",
			false,
		},
		{
			"empty text",
			&Slack{},
			args{
				ie: slack.RichTextSectionElement(slack.NewRichTextSectionLinkElement("https://example.com", "", nil)),
			},
			"<a href=\"https://example.com\">https://example.com</a>",
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.s.rtseLink(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("rtseLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rtseLink() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("rtseLink() = %v, want1 %v", got1, tt.want1)
			}
		})
	}
}

func TestSlack_rtseUserGroup(t *testing.T) {
	type args struct {
		ie slack.RichTextSectionElement
	}
	tests := []struct {
		name    string
		s       *Slack
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"user group",
			&Slack{},
			args{
				ie: slack.NewRichTextSectionUserGroupElement("W12345678"),
			},
			`<div class="slack-rich-text-section-user-group"><@W12345678></div>`,
			``,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.s.rtseUserGroup(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slack.rtseUserGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Slack.rtseUserGroup() = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Slack.rtseUserGroup() = %v, want1 %v", got1, tt.want1)
			}
		})
	}
}

func TestSlack_rtseColor(t *testing.T) {
	type args struct {
		ie slack.RichTextSectionElement
	}
	tests := []struct {
		name    string
		s       *Slack
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			"color block",
			&Slack{},
			args{
				ie: slack.NewRichTextSectionColorElement("#ff0000"),
			},
			`<span style="color: #ff0000;">`,
			`</span>`,
			false,
		},
		{
			"real",
			&Slack{},
			args{
				ie: load[*slack.RichTextSectionColorElement](t, `{"type": "color","value": "#2475D9"}`),
			},
			`<span style="color: #2475D9;">`,
			`</span>`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.s.rtseColor(tt.args.ie)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slack.rtseColor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Slack.rtseColor() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Slack.rtseColor() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMbtRichText(t *testing.T) {
	const colorfull = `{
  "type": "rich_text",
  "block_id": "arhEv",
  "elements": [
    {
      "type": "rich_text_section",
      "elements": [
        {
          "type": "color",
          "value": "#2475D9"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#FC6215"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#FFFFFF"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#0A4B8C"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#FD8B24"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#FFFFFF"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#46BE6E"
        },
        {
          "type": "text",
          "text": ","
        },
        {
          "type": "color",
          "value": "#EE2436"
        }
      ]
    }
  ]
}`
	t.Run("colorfull", func(t *testing.T) {
		s := &Slack{}
		m := load[*slack.RichTextBlock](t, colorfull)
		got, _, err := s.mbtRichText(m)
		if err != nil {
			t.Errorf("Slack.rtseSection() error = %v", err)
			return
		}
		if got != `<span style="color: #2475D9;">,<span style="color: #FC6215;">,<span style="color: #FFFFFF;">,<span style="color: #0A4B8C;">,<span style="color: #FD8B24;">,<span style="color: #FFFFFF;">,<span style="color: #46BE6E;">,<span style="color: #EE2436;"></span></span></span></span></span></span></span></span>` {
			t.Errorf("Slack.rtseSection() = %v", got)
		}
	})
}
