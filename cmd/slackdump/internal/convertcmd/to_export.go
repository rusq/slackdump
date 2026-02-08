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

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/source"
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
	sttFn, ok := cflg.outStorageType.Func()
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
		convert.WithTrgFileLoc(sttFn),
		convert.WithLogger(cfg.Log),
	)
	if err := cvt.Convert(ctx); err != nil {
		return err
	}

	slog.Info("converted", "source", src, "target", trg)
	return nil
}
