package convertcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"text/tabwriter"
	"time"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
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

	slog.Info("output", "directory", dir)
	cd, err := chunk.OpenDir(dir)
	if err != nil {
		return err
	}
	erc := dirproc.NewERC(cd, cfg.Log)
	defer erc.Close()

	srcdb, err := source.OpenDatabase(ctx, src)
	if err != nil {
		return err
	}
	defer srcdb.Close()

	remove = false // init succeeded

	if err := srcdb.ToChunk(ctx, erc, cflg.sessionID); err != nil {
		if errors.Is(err, dbproc.ErrInvalidSessionID) {
			sess, err := srcdb.Sessions(ctx)
			if err != nil {
				return errors.New("no sessions found")
			}
			printSessions(os.Stderr, sess)
		}
		return err
	}
	return nil
}

func printSessions(w io.Writer, sessions []repository.Session) {
	const layout = time.DateTime
	tz := time.Local
	fmt.Fprintf(w, "\nSessions in the data base (timezone: %s):\n\n", tz)
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintln(tw, "  ID  \tDate\tComplete\tMode")
	fmt.Fprintln(tw, "------\t----\t--------\t----")
	for _, s := range sessions {
		fmt.Fprintf(tw, "%6d\t%s\t%v\t%s\n", s.ID, s.CreatedAt.In(tz).Format(time.DateTime), s.Finished, s.Mode)
	}
	fmt.Fprintln(tw)
}
