package export

import (
	"context"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/export"
	"github.com/rusq/slackdump/v3/internal/structures"
)

func exportV2(ctx context.Context, sess *slackdump.Session, fs fsadapter.FS, list *structures.EntityList, flags exportFlags) error {
	config := export.Config{
		Oldest:      time.Time(flags.Oldest),
		Latest:      time.Time(flags.Latest),
		Logger:      cfg.Log,
		List:        list,
		Type:        export.ExportType(flags.ExportStorageType),
		MemberOnly:  flags.MemberOnly,
		ExportToken: flags.ExportToken,
	}
	exp := export.New(sess, fs, config)
	return exp.Run(ctx)
}
