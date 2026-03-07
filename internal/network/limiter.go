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

package network

import (
	"time"

	"golang.org/x/time/rate"
)

// Tier represents rate limit Tier:
// https://api.slack.com/docs/rate-limits
type Tier int

const (
	// base throttling defined in events per minute
	NoTier Tier = 6000 // no Tier is applied

	// Tier1 Tier = 1
	Tier2 Tier = 20
	Tier3 Tier = 50
	Tier4 Tier = 100
)

// NewLimiter returns throttler with rateLimit requests per minute.
// optionally caller may specify the boost
func NewLimiter(t Tier, burst uint, boost int) *rate.Limiter {
	l := rate.NewLimiter(rate.Every(every(t, boost)), int(burst))
	return l
}

func every(t Tier, boost int) time.Duration {
	return time.Minute / time.Duration(int(t)+int(boost))
}
