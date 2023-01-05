package list

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdListChannels = &base.Command{
	Run:        listChannels,
	UsageLine:  "slackdump list channels [flags] [filename]",
	PrintFlags: true,
	FlagMask:   cfg.OmitDownloadFlag,
	Short:      "list workspace channels",
	Long: base.Render(`
# List Channels Command

Lists all visible channels for the currently logged in user.  The list
includes all public and private channels, groups, and private messages (DMs),
including archived ones.

Please note that it may take a while to retrieve all channels, if your
workspace has lots of them.
`),
	RequireAuth: true,
}

func listChannels(ctx context.Context, cmd *base.Command, args []string) error {
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = fmt.Sprintf("channels-%s.json", sess.Info().TeamID)
		if len(args) > 0 {
			filename = args[0]
		}
		a, err := sess.GetChannels(ctx)
		return a, filename, err
	}); err != nil {
		return err
	}

	return nil
}
