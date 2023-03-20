package export

import (
	"context"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/structures"
)

func exportV2(ctx context.Context, sess *slackdump.Session, list *structures.EntityList, options export.Config) error {
	exp := export.New(sess, options)
	return exp.Run(ctx)
}
