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

package client

import (
	"context"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/edge"
)

// NewEdge returns a new *Client that is guaranteed to have an edge (enterprise)
// connection.  Use c.Edge() to obtain the underlying *edge.Client.
func NewEdge(ctx context.Context, prov auth.Provider, opts ...Option) (*Client, error) {
	scl, wi, err := newSlackClient(ctx, prov)
	if err != nil {
		return nil, err
	}
	ecl, err := edge.NewWithInfo(wi, prov)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client: scl,
		edge:   ecl,
		wi:     wi,
	}, nil
}
