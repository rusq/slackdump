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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/source"
)

// toDatabase converts the source to the database format.
func toDatabase(ctx context.Context, src, trg string, cflg convertflags) error {
	// detect source type
	st, err := source.Type(src)
	if err != nil {
		return err
	}

	switch {
	case st == source.FUnknown:
		return ErrSource
	case st.Has(source.FChunk):
		return dbConvertFast(ctx, src, trg, cflg)
	default:
		return dbConvert(ctx, src, trg, cflg)
	}
}

// dbConvertFast converts the chunk source to the database format.
func dbConvertFast(ctx context.Context, src, trg string, cflg convertflags) error {
	cd, err := chunk.OpenDir(src)
	if err != nil {
		return err
	}
	defer cd.Close()
	dsrc := source.OpenChunkDir(cd, true)
	defer dsrc.Close()

	trg = cfg.StripZipExt(trg)
	if err := chunk2db(ctx, dsrc, trg, cflg); err != nil {
		return err
	}

	if cflg.includeFiles && dsrc.Files().Type() != source.STnone {
		slog.Info("Copying files...")
		if err := copyfiles(filepath.Join(trg, chunk.UploadsDir), dsrc.Files().FS()); err != nil {
			return err
		}
	}
	if cflg.includeAvatars && dsrc.Avatars().Type() != source.STnone {
		slog.Info("Copying avatars...")
		if err := copyfiles(filepath.Join(trg, chunk.AvatarsDir), dsrc.Avatars().FS()); err != nil {
			return err
		}
	}
	return nil
}

// chunk2db converts the chunk source to the database format, it creates the
// database in the directory dir.
func chunk2db(ctx context.Context, src *source.ChunkDir, dir string, cflg convertflags) error {
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

	slog.Info("output", "database", filepath.Join(dir, source.DefaultDBFile))

	// create a new database
	wconn, si, err := bootstrap.Database(dir, "convert")
	if err != nil {
		return err
	}
	defer wconn.Close()

	dbp, err := dbase.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer dbp.Close()

	txx, err := wconn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer txx.Rollback()

	enc := &encoder{dbp: dbp, tx: txx}
	if err := src.ToChunk(ctx, enc, cflg.sessionID); err != nil {
		return err
	}
	if err := txx.Commit(); err != nil {
		return err
	}

	remove = false
	return nil
}

// encoder implements the chunk.Encoder around the unsafe database insert.
// It operates in a single transaction tx.
type encoder struct {
	dbp *dbase.DBP
	tx  *sqlx.Tx
}

func (e *encoder) Encode(ctx context.Context, ch *chunk.Chunk) error {
	_, err := e.dbp.UnsafeInsertChunk(ctx, e.tx, ch)
	return err
}

func dbConvert(ctx context.Context, src, dir string, cflg convertflags) error {
	s, err := source.Load(ctx, src)
	if err != nil {
		return err
	}
	defer s.Close()

	dir = cfg.StripZipExt(dir)

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
	fsa := fsadapter.NewDirectory(dir)
	defer fsa.Close()

	// create a new database
	wconn, si, err := bootstrap.Database(dir, "convert")
	if err != nil {
		return err
	}
	defer wconn.Close()

	dbp, err := dbase.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer dbp.Close()

	txx, err := wconn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer txx.Rollback()

	enc := &encoder{dbp: dbp, tx: txx}

	conv := convert.NewSourceEncoder(
		s,
		fsa,
		enc,
		convert.WithLogger(cfg.Log),
		convert.WithIncludeFiles(cflg.includeFiles),
		convert.WithIncludeAvatars(cflg.includeAvatars),
		convert.WithTrgFileLoc(source.MattermostFilepath),
	)

	if err := conv.Convert(ctx); err != nil {
		return err
	}
	if err := txx.Commit(); err != nil {
		return err
	}

	remove = false
	return nil
}
