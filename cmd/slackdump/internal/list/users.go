package list

import (
	"context"

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
	Long: base.Render(`
# List Users

List users lists workspace users in the desired format.` +
		sectListFormat,
	),
	RequireAuth: true,
}

func listUsers(ctx context.Context, cmd *base.Command, args []string) error {
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, listType)
		if len(args) > 0 {
			filename = args[0]
		}
		return sess.Users, filename, nil
	}); err != nil {
		return err
	}
	return nil
}

func wizUsers(ctx context.Context, cmd *base.Command, args []string) error {
	return wizard(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, listType)
		return sess.Users, filename, nil
	})
}
