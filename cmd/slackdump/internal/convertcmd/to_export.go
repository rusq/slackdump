package convertcmd

import (
	"context"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/internal/source"
)

func toExport(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}

	if st == source.FUnknown {
		return ErrSource
	}

	fsa, err := fsadapter.New(trg)
	if err != nil {
		return err
	}
	defer fsa.Close()

	// output storage
	sttFn, ok := fileproc.StorageTypeFuncs[cflg.outStorageType]
	if !ok {
		return ErrStorage
	}

	var (
		includeFiles   = cflg.includeFiles && st.Has(source.FMattermost)
		includeAvatars = cflg.includeAvatars && st.Has(source.FAvatars)
	)

	s, err := source.Load(ctx, src)
	if err != nil {
		return err
	}

	cvt := convert.NewToExport(
		s,
		fsa,
		convert.WithIncludeFiles(includeFiles),
		convert.WithIncludeAvatars(includeAvatars),
		convert.WithSrcFileLoc(sttFn),
		convert.WithTrgFileLoc(sttFn),
		convert.WithLogger(cfg.Log),
	)
	if err := cvt.Convert(ctx); err != nil {
		return err
	}

	return nil
}
