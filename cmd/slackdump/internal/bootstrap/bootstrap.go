// Package bootstrap contains some initialisation functions that are shared
// between main some other top level commands, i.e. wizard.
package bootstrap

import (
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
)

func CurrentWsp() string {
	if current, err := workspace.Current(cfg.CacheDir(), cfg.Workspace); err == nil {
		return current
	}
	return "<not set>"
}
