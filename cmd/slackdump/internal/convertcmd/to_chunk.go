// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package convertcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/directory"
	"github.com/rusq/slackdump/v4/source"
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

	srcdb, err := source.OpenDatabase(ctx, src)
	if err != nil {
		return err
	}
	defer srcdb.Close()

	trg = cfg.StripZipExt(trg)

	if err := db2chunk(ctx, srcdb, trg, cflg); err != nil {
		return err
	}
	if cflg.includeFiles && srcdb.Files().Type() != source.STnone {
		slog.Info("Copying files...")
		if err := copyfiles(filepath.Join(trg, chunk.UploadsDir), srcdb.Files().FS()); err != nil {
			return err
		}
	}
	if cflg.includeAvatars && srcdb.Avatars().Type() != source.STnone {
		slog.Info("Copying avatars...")
		if err := copyfiles(filepath.Join(trg, chunk.AvatarsDir), srcdb.Avatars().FS()); err != nil {
			return err
		}
	}
	return nil
}

// db2chunk converts the database to the chunk format, writing to the directory dir.
func db2chunk(ctx context.Context, src *source.Database, dir string, cflg convertflags) error {
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
	erc := directory.NewERC(cd, cfg.Log)
	defer erc.Close()

	if err := src.ToChunk(ctx, erc, cflg.sessionID); err != nil {
		if errors.Is(err, dbase.ErrInvalidSessionID) {
			sess, err := src.Sessions(ctx)
			if err != nil {
				return errors.New("no sessions found")
			}
			printSessions(os.Stderr, sess)
		}
		return err
	}
	remove = false
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
		fmt.Fprintf(tw, "%6d\t%s\t%v\t%s\n", s.ID, s.CreatedAt.In(tz).Format(layout), s.Finished, s.Mode)
	}
	fmt.Fprintln(tw)
}
