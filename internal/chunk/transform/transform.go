package transform

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

var ErrClosed = errors.New("transformer is closed")

// Converter is the interface that defines a set of methods for transforming
// chunks to some output format.
type Converter interface {
	// Convert should convert the chunk to the Converters' output format.
	Convert(ctx context.Context, id chunk.FileID) error
}
