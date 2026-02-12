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

// strategy is an interface that defines the strategy for selecting the next
// item.
type strategy interface {
	// next returns the next item in the pool.
	next() int
}

// roundRobin implements the round-robin strategy.
type roundRobin struct {
	// total is the total number of items in the pool.
	total int
	// i is the current item index.
	i int
}

// newRoundRobin creates a new round-robin strategy with the given total number
// of items.
func newRoundRobin(total int) *roundRobin {
	return &roundRobin{total: total}
}

func (r *roundRobin) next() int {
	r.i = (r.i + 1) % r.total
	return r.i
}
