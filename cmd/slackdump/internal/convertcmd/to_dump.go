package convertcmd

import (
	"context"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/source"
)

func toDump(ctx context.Context, srcpath, trgloc string, cflg convertflags) error {
	src, err := source.Load(ctx, srcpath)
	if err != nil {
		return err
	}
	defer src.Close()

	fsa, err := fsadapter.New(trgloc)
	if err != nil {
		return err
	}
	defer fsa.Close()

	conv, err := transform.NewDumpConverter(fsa, src, transform.DumpWithLogger(cfg.Log))
	if err != nil {
		return err
	}

	channels, err := src.Channels(ctx)
	if err != nil {
		return err
	}
	for _, c := range channels {
		if err := conv.Convert(ctx, chunk.ToFileID(c.ID, "", false)); err != nil {
			return err
		}
	}

	return nil
}
