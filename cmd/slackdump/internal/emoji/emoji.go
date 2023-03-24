package emoji

import (
	"context"
	"fmt"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/emoji"
)

var CmdEmoji = &base.Command{
	Run:         run,
	Wizard:      wizard,
	UsageLine:   "slackdump emoji [flags]",
	Short:       "download workspace emojis",
	Long:        "", // TODO: add long description
	FlagMask:    cfg.OmitDownloadFlag | cfg.OmitConfigFlag,
	RequireAuth: true,
	PrintFlags:  true,
}

// emoji specific flags
var (
	ignoreErrors bool
)

func init() {
	CmdEmoji.Flag.BoolVar(&ignoreErrors, "ignore-errors", true, "ignore download errors (skip failed emojis)")
}

func run(ctx context.Context, cmd *base.Command, args []string) error {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return fmt.Errorf("auth error: %s", err)
	}

	fs, err := fsadapter.New(cfg.BaseLocation)
	if err != nil {
		return err
	}
	defer fs.Close()

	sess, err := slackdump.New(ctx, prov, slackdump.WithFilesystem(fs), slackdump.WithLogger(dlog.FromContext(ctx)))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}

	if err := emoji.DlFS(ctx, sess, fs, ignoreErrors); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}
	return nil
}
