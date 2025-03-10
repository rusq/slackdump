package convertcmd

import (
	"context"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/source"
)

func toDatabase(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}
	if !st.Has(source.FChunk) {
		return ErrSource
	}
	cd, err := chunk.OpenDir(src)
	if err != nil {
		return err
	}
	defer cd.Close()

	trg = cfg.StripZipExt(trg)
	if err := os.MkdirAll(trg, 0o755); err != nil {
		return err
	}
	remove := true
	defer func() {
		// remove on failed conversion
		if remove {
			_ = os.RemoveAll(trg)
		}
	}()
	wconn, si, err := bootstrap.Database(trg, "convert")
	if err != nil {
		return err
	}
	defer wconn.Close()

	dbp, err := dbproc.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer dbp.Close()

	txx, err := wconn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer txx.Rollback()

	remove = false // init succeeded
	if err := cd.WalkSync(func(name string, f *chunk.File, err error) error {
		if err != nil {
			return err
		}
		if err := f.ForEach(func(ch *chunk.Chunk) error {
			_, err := dbp.UnsafeInsertChunk(ctx, txx, ch)
			return err
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err := txx.Commit(); err != nil {
		return err
	}
	return nil
}
