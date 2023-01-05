package list

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/convert/format"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

var CmdListUsers = &base.Command{
	Run:        listUsers,
	UsageLine:  "slackdump list users [flags]",
	PrintFlags: true,
	FlagMask:   cfg.OmitDownloadFlag,
	Short:      "list workspace users",
	Long: `
List users lists workspace users in the desired format.
`,
	RequireAuth: true,
}

func listUsers(ctx context.Context, cmd *base.Command, args []string) error {
	var filename string
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		filename = fmt.Sprintf("users-%s.json", sess.Info().TeamID)
		a, err := sess.GetUsers(ctx)
		return a, filename, err
	}); err != nil {
		return err
	}

	if listType == format.CUnknown {
		dlog.FromContext(ctx).Printf("users saved to %q\n", filepath.Join(cfg.BaseLoc, filename))
	}
	return nil
}
