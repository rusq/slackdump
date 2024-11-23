package emoji

import (
	"context"
	"fmt"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/emoji/emojidl"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/edge"
)

var CmdEmoji = &base.Command{
	Run:         run,
	UsageLine:   "slackdump emoji [flags]",
	Short:       "download workspace emojis",
	Long:        "", // TODO: add long description
	FlagMask:    cfg.OmitDownloadFlag | cfg.OmitConfigFlag,
	RequireAuth: true,
	PrintFlags:  true,
}

type options struct {
	ignoreErrors bool
	fullInfo     bool
}

// emoji specific flags
var cmdFlags = options{
	ignoreErrors: false,
}

func init() {
	CmdEmoji.Wizard = wizard
	CmdEmoji.Flag.BoolVar(&cmdFlags.ignoreErrors, "ignore-errors", true, "ignore download errors (skip failed emojis)")
	CmdEmoji.Flag.BoolVar(&cmdFlags.fullInfo, "full-info", true, "fetch emojis using Edge API to get full emoji information, including usernames")
}

func run(ctx context.Context, cmd *base.Command, args []string) error {
	fsa, err := fsadapter.New(cfg.Output)
	if err != nil {
		return err
	}
	defer fsa.Close()

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	if cmdFlags.fullInfo {
		return runEdge(ctx, fsa, prov)
	} else {
		return runLegacy(ctx, fsa)
	}
}

func runLegacy(ctx context.Context, fsa fsadapter.FS) error {
	sess, err := bootstrap.SlackdumpSession(ctx, slackdump.WithFilesystem(fsa))
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	return emojidl.DlFS(ctx, sess, fsa, cmdFlags.ignoreErrors)
}

func runEdge(ctx context.Context, fsa fsadapter.FS, prov auth.Provider) error {
	sess, err := edge.New(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer sess.Close()

	if err := emojidl.DlEdgeFS(ctx, sess, fsa, cmdFlags.ignoreErrors); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}
	return nil
}
