package renderer

import (
	"testing"

	"github.com/rusq/slack"
)

func TestAuditXSS(t *testing.T) {
	s := &Slack{}
	img := "https://x/y.png"
	h, _, _ := s.metImage(&slack.ImageBlockElement{ImageURL: &img, AltText: `x" onerror="alert(1)`})
	t.Log(h)
	h, _, _ = s.rtseLink(&slack.RichTextSectionLinkElement{URL: `javascript:alert(1)`, Text: "click"})
	t.Log(h)
	h, _, _ = s.rtseLink(&slack.RichTextSectionLinkElement{URL: `https://a"><script>alert(1)</script>`})
	t.Log(h)
	h, _, _ = s.rtseColor(&slack.RichTextSectionColorElement{Value: `red;"><script>alert(1)</script>`})
	t.Log(h)
}
