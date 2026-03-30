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

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/convert"
	"github.com/rusq/slackdump/v4/source"
)

func toHTML(ctx context.Context, srcpath, trgdir string, _ convertflags) error {
	st, err := source.Type(srcpath)
	if err != nil {
		return err
	}
	if st == source.FUnknown {
		return ErrSource
	}

	src, err := source.Load(ctx, srcpath)
	if err != nil {
		return err
	}
	defer src.Close()

	conv := convert.NewToHTML(src, fsadapter.NewDirectory(trgdir), convert.WithLogger(cfg.Log))
	if err := conv.Convert(ctx); err != nil {
		return err
	}

	cfg.Log.InfoContext(ctx, "converted", "source", srcpath, "target", trgdir)
	return nil
}
