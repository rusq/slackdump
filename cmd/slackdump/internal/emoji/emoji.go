package emoji

import (
	"context"
	"fmt"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/emoji/emojidl"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
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
	fsa, err := fsadapter.New(cfg.Output)
	if err != nil {
		return err
	}
	defer fsa.Close()

	sess, err := cfg.SlackdumpSession(ctx, slackdump.WithFilesystem(fsa))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}

	if err := emojidl.DlFS(ctx, sess, fsa, ignoreErrors); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}
	return nil
}
