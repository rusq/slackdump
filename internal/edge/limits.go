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
package edge

import (
	"time"

	"golang.org/x/time/rate"
)

type tier struct {
	// once every
	t time.Duration
	// burst
	b int
}

func (t tier) limiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(t.t), t.b)
}

var (
	// tier1 = tier{t: 1 * time.Minute, b: 2}
	// tier2 = tier{t: 3 * time.Second, b: 3}
	tier2boost = tier{t: 300 * time.Millisecond, b: 5}
	tier3      = tier{t: 1200 * time.Millisecond, b: 4}
	// tier4      = tier{t: 60 * time.Millisecond, b: 5}
)
