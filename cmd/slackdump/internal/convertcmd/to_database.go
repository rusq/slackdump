package convertcmd

import (
	"context"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/source"
)

// toDatabase converts the source to the database format.
func toDatabase(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}

	// currently only chunk format is supported for the source.
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

	// writer connection
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
	enc := &encoder{dbp: dbp, tx: txx}
	if err := cd.ToChunk(ctx, enc, cflg.sessionID); err != nil {
		return err
	}
	if err := txx.Commit(); err != nil {
		return err
	}
	return nil
}

type encoder struct {
	dbp *dbproc.DBP
	tx  *sqlx.Tx
}

func (e *encoder) Encode(ctx context.Context, ch *chunk.Chunk) error {
	_, err := e.dbp.UnsafeInsertChunk(ctx, e.tx, ch)
	return err
}
