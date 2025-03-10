package convertcmd

import (
	"context"
	"os"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/source"
)

func toChunk(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}
	if !st.Has(source.FDatabase) {
		return ErrSource
	}
	wconn, _, err := bootstrap.Database(src, "convert")
	if err != nil {
		return err
	}
	defer wconn.Close()

	dir := cfg.StripZipExt(trg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	remove := true
	defer func() {
		// remove on failed conversion
		if remove {
			_ = os.RemoveAll(dir)
		}
	}()

	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	erc := dirproc.NewERC(cd, cfg.Log)
	defer erc.Close()

	remove = false // init succeeded

	// TODO: how to find the session id?
	ch := dbproc.Chunker{SessionID: 1}
	if err := ch.ToChunk(ctx, wconn, erc); err != nil {
		return err
	}
	return nil
}
