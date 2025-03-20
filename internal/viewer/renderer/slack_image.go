package renderer

import (
	"fmt"

	"github.com/rusq/slack"
)

func (*Slack) mbtImage(ib slack.Block) (string, string, error) {
	b, ok := ib.(*slack.ImageBlock)
	if !ok {
		return "", "", NewErrIncorrectType(&slack.ImageBlock{}, ib)
	}
	return elFigure(
		blockTypeClass[slack.MBTImage],
		fmt.Sprintf(
			`<img src="%[1]s" alt="%[2]s"><figcaption class="slack-image-caption">%[2]s</figcaption>`,
			b.ImageURL, b.AltText,
		),
	), "", nil
}
