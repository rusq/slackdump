package list

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdListUsers = &base.Command{
	Run:        listUsers,
	Wizard:     wizUsers,
	UsageLine:  "slackdump list users [flags] [filename]",
	PrintFlags: true,
	FlagMask:   cfg.OmitDownloadFlag,
	Short:      "list workspace users",
	Long: fmt.Sprintf(`
# List Users

List users lists workspace users in the desired format.

Users are cached for %v.  To disable caching, use '-no-user-cache' flag and
'-user-cache-retention' flag to control the caching behaviour.
`+
		sectListFormat, cfg.UserCacheRetention),
	RequireAuth: true,
}

func listUsers(ctx context.Context, cmd *base.Command, args []string) error {
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, ".json")
		if len(args) > 0 {
			filename = args[0]
		}
		users, err := sess.GetUsers(ctx)
		return users, filename, err
	}); err != nil {
		return err
	}
	return nil
}

func wizUsers(ctx context.Context, cmd *base.Command, args []string) error {
	return wizard(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, ".json")
		users, err := sess.GetUsers(ctx)
		return users, filename, err
	})
}
