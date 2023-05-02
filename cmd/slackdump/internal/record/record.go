package record

import (
	"context"
	_ "embed"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

//go:embed assets/record.md
var mdRecord string

var CmdRecord = &base.Command{
	Run:         RunRecord,
	UsageLine:   "slackdump record [link1[ link 2[ link N]]]",
	Short:       "record the dump of the workspace or individual conversations",
	Long:        mdRecord,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	RequireAuth: true,
	PrintFlags:  true,
}

func RunRecord(ctx context.Context, cmd *base.Command, args []string) error {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	_ = prov
	return nil
}
