package transform

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

type Interface interface {
	// Transform transforms the conversation files from the temporary directory
	// with state files and chunk records.
	Transform(ctx context.Context, basePath string, st *state.State) error
}
