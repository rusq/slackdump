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

package slackdump

// In this file: slackdump config.

import (
	"time"

	"github.com/rusq/slackdump/v4/internal/network"
)

// Config is the option set for the Session.
type config struct {
	limits          network.Limits
	dumpFiles       bool          // will we save the conversation files?
	cacheRetention  time.Duration // how long to keep the cache (user, etc.)
	forceEnterprise bool          // force enterprise workspace
}

// DefOptions is the default options used when initialising slackdump instance.
var defConfig = config{
	limits:         network.DefLimits,
	dumpFiles:      false,
	cacheRetention: 60 * time.Minute,
}
