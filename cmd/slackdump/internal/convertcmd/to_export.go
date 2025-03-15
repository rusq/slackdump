package convertcmd

import (
	"context"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
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
	} else if st.Has(source.FExport) {
		return ErrMeaningless
	}

	fsa, err := fsadapter.New(trg)
	if err != nil {
		return err
	}
	defer fsa.Close()

	// output storage
	sttFn, ok := source.StorageTypeFuncs[cflg.outStorageType]
	if !ok {
		return ErrStorage
	}

	s, err := source.Load(ctx, src)
	if err != nil {
		return err
	}
	defer s.Close()

	var (
		includeFiles   = cflg.includeFiles && s.Files().Type() != source.STnone
		includeAvatars = cflg.includeAvatars && s.Avatars().Type() != source.STnone
	)

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
