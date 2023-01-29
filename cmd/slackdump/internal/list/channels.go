package list

import (
	"context"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/types"
)

var CmdListChannels = &base.Command{
	Run:        listChannels,
	UsageLine:  "slackdump list channels [flags] [filename]",
	PrintFlags: true,
	FlagMask:   cfg.OmitDownloadFlag,
	Short:      "list workspace channels",
	Long: `
# List Channels Command

Lists all visible channels for the currently logged in user.  The list
includes all public and private channels, groups, and private messages (DMs),
including archived ones.

Please note that it may take a while to retrieve all channels, if your
workspace has lots of them.
` + sectListFormat,

	RequireAuth: true,
}

func listChannels(ctx context.Context, cmd *base.Command, args []string) error {
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("channels", sess.Info().TeamID, listType)
		if len(args) > 0 {
			filename = args[0]
		}

		cc, ok := maybeLoadChanCache(cfg.CacheDir(), sess.Info().Team)
		if ok {
			return cc, filename, nil
		}
		cc, err := sess.GetChannels(ctx)
		return cc, filename, err
	}); err != nil {
		return err
	}

	return nil
}

var chanCacheOpts = slackdump.CacheOptions{
	Disabled: false,
}

func maybeLoadChanCache(cacheDir string, teamID string) (types.Channels, bool) {
	m, err := cache.NewManager(cacheDir)
	if err != nil {
		return nil, false
	}
	cc, err := m.LoadChannels(teamID, chanCacheOpts.MaxAge)
	if err != nil {
		return nil, false
	}
	return cc, true
}
