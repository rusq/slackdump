package convertcmd

import (
	"context"
	"errors"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/source"
)

var ErrMeaningless = errors.New("meaningless conversion")

func toDump(ctx context.Context, srcpath, trgloc string, cflg convertflags) error {
	st, err := source.Type(srcpath)
	if err != nil {
		return err
	}
	if st == source.FUnknown {
		return ErrSource
	} else if st.Has(source.FDump) {
		return ErrMeaningless
	}
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

	filesEnabled := cflg.includeFiles && src.Files().Type() != source.STnone

	conv := convert.NewToDump(src, fsa, convert.DumpWithIncludeFiles(filesEnabled), convert.DumpWithLogger(cfg.Log))

	if err := conv.Convert(ctx); err != nil {
		return err
	}

	cfg.Log.InfoContext(ctx, "converted", "source", srcpath, "target", trgloc)
	return nil
}
