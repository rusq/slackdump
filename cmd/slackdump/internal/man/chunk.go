package man

import (
	_ "embed"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
)

//go:embed assets/chunk.md
var mdChunk string

var Chunk = &base.Command{
	UsageLine: "slackdump chunk",
	Short:     "chunk file format specification",
	Long:      mdChunk,
}
