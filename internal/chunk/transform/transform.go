package transform

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

var ErrClosed = errors.New("transformer is closed")

type Interface interface {
	// Transform transforms the conversation files from the temporary directory
	// with state files and chunk records.
	Transform(ctx context.Context, basePath string, st *state.State) error
}
