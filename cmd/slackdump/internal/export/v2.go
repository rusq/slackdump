package export

import (
	"context"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/structures"
)

func exportV2(ctx context.Context, sess *slackdump.Session, fs fsadapter.FS, list *structures.EntityList, options export.Config) error {
	exp := export.New(sess, fs, options)
	return exp.Run(ctx)
}
