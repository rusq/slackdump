package renderer

import (
	"fmt"

	"github.com/rusq/slack"
)

func (*Slack) mbtImage(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.ImageBlock)
	if !ok {
		return "", NewErrIncorrectType(&slack.ImageBlock{}, ib)
	}
	return fmt.Sprintf(
		`<figure class="slack-image-block"><img src="%[1]s" alt="%[2]s"><figcaption class="slack-image-caption">%[2]s</figcaption></figure>`,
		b.ImageURL, b.AltText,
	), nil
}
