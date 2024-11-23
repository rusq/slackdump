package slackdump

import (
	"context"

	"github.com/rusq/slack"
)

func (s *Session) DumpEmojis(ctx context.Context) (map[string]string, error) {
	emoji, err := s.client.GetEmojiContext(ctx)
	if err != nil {
		return nil, err
	}
	return emoji, nil
}

func (s *Session) DumpEmojisAdmin(ctx context.Context) (map[string]slack.Emoji, error) {
	var ret = make(map[string]slack.Emoji, 100)

	p := slack.AdminEmojiListParameters{Cursor: "", Limit: 100}
	for {
		emoji, next, err := s.client.AdminEmojiList(ctx, p)
		if err != nil {
			return nil, err
		}
		for k, v := range emoji {
			ret[k] = v
		}
		if next == "" {
			break
		}
		p.Cursor = next
	}
	return ret, nil
}
