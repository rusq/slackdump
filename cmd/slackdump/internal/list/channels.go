package list

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
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
		ctx, task := trace.NewTask(ctx, "listChannels")
		defer task.End()

		var filename = makeFilename("channels", sess.Info().TeamID, listType)
		if len(args) > 0 {
			filename = args[0]
		}
		teamID := sess.Info().TeamID
		cc, ok := maybeLoadChanCache(cfg.CacheDir(), teamID)
		if ok {
			// cache hit
			trace.Logf(ctx, "cache hit", "teamID=%s", teamID)
			return cc, filename, nil
		}
		// cache miss, load from API
		trace.Logf(ctx, "cache miss", "teamID=%s", teamID)
		cc, err := sess.GetChannels(ctx)
		if err != nil {
			return nil, "", err
		}
		if err := saveCache(cfg.CacheDir(), teamID, cc); err != nil {
			// warn, but don't fail
			dlog.FromContext(ctx).Printf("failed to save cache: %v", err)
		}
		return cc, filename, nil
	}); err != nil {
		return err
	}

	return nil
}

var chanCacheOpts = slackdump.CacheConfig{
	Disabled: false,
	MaxAge:   0 * time.Minute,
	Filename: "channels.json",
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

func saveCache(cacheDir, teamID string, cc types.Channels) error {
	m, err := cache.NewManager(cacheDir)
	if err != nil {
		return err
	}
	return m.SaveChannels(teamID, cc)
}
