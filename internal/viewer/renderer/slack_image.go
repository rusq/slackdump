package renderer

import (
	"fmt"

	"github.com/rusq/slack"
)

func mbtImage(ib slack.Block) (string, error) {
	b, ok := ib.(*slack.ImageBlock)
	if !ok {
		return "", NewErrIncorrectType(&slack.ImageBlock{}, ib)
	}
	return fmt.Sprintf(`<img src="%s" alt="%s">`, b.ImageURL, b.AltText), nil
}
