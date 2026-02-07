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
package directory

import (
	"context"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

// Search is the search results directory processor.  The results are written
// to "search.json.gz" file in the chunk directory.
type Search struct {
	*dirproc

	subproc processor.Filer

	recordFiles bool
}

// NewSearch creates a new search processor.
func NewSearch(dir *chunk.Directory, filer processor.Filer) (*Search, error) {
	p, err := newDirProc(dir, chunk.FSearch)
	if err != nil {
		return nil, err
	}
	return &Search{
		dirproc: p,
		subproc: filer,
	}, nil
}

func (s *Search) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	if err := s.subproc.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	if !s.recordFiles {
		return nil
	}
	if err := s.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	return nil
}
