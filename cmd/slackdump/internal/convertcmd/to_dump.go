package convertcmd

import (
	"context"
	"errors"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/internal/source"
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

	filesEnabled := cflg.includeFiles && (st.Has(source.FMattermost))

	conv := convert.NewToDump(src, fsa, convert.DumpWithIncludeFiles(filesEnabled), convert.DumpWithLogger(cfg.Log))

	return conv.Convert(ctx)
}
