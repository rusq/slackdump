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

package stream

import (
	"context"
	"errors"
	"runtime/trace"

	"github.com/rusq/slackdump/v4/processor"
)

var ErrNoEdgeClient = errors.New("saved items require an edge client, see stream.OptEdgeClient")

// SavedItems fetches the current user's "Later" items via [OptEdgeClient].
func (cs *Stream) SavedItems(ctx context.Context, proc processor.SavedItemsCollector) error {
	ctx, task := trace.NewTask(ctx, "SavedItems")
	defer task.End()

	if cs.edge == nil {
		return ErrNoEdgeClient
	}
	items, err := cs.edge.SavedList(ctx)
	if err != nil {
		return err
	}
	return proc.SavedItems(ctx, items)
}
