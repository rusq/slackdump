package emoji

import (
	"context"

	"github.com/rusq/dlog"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/app/emoji"
)

var CmdEmoji = &base.Command{
	Run:         runEmoji,
	Wizard:      func(context.Context, *base.Command, []string) error { panic("not implemented") },
	UsageLine:   "slackdump emoji [flags]",
	Short:       "download workspace emojis",
	Long:        "",
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

func runEmoji(ctx context.Context, cmd *base.Command, args []string) {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		dlog.Printf("auth error: %s", err)
		return
	}
	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.SlackOptions)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		dlog.Printf("application error: %s", err)
		return
	}
	if err := emoji.Dl(ctx, sess, cfg.BaseLoc, ignoreErrors); err != nil {
		base.SetExitStatus(base.SApplicationError)
		dlog.Printf("application error: %s", err)
		return
	}
}
