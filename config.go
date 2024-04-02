package slackdump

// In this file: slackdump config.

import (
	"time"

	"github.com/rusq/slackdump/v3/internal/network"
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
