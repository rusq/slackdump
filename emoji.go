package slackdump

import (
	"context"
)

func (s *Session) DumpEmojis(ctx context.Context) (map[string]string, error) {
	emoji, err := s.client.GetEmojiContext(ctx)
	if err != nil {
		return nil, err
	}
	return emoji, nil
}
