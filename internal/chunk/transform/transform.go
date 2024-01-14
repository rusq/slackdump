package transform

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/slack-go/slack"
)

var ErrClosed = errors.New("transformer is closed")

type Converter interface {
	Convert(ctx context.Context, id chunk.FileID) error
}

type UserConverter interface {
	Converter
	SetUsers([]slack.User)
	HasUsers() bool
}
