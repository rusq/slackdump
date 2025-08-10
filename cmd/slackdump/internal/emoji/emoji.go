package emoji

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/emoji/emojidl"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/client"
)

//go:embed assets/emoji.md
var emojiMD string

var CmdEmoji = &base.Command{
	Run:         run,
	UsageLine:   "slackdump emoji [flags]",
	Short:       "download workspace emoticons ಠ_ಠ",
	Long:        emojiMD,
	FlagMask:    cfg.OmitAll &^ cfg.OmitAuthFlags &^ cfg.OmitOutputFlag &^ cfg.OmitWorkspaceFlag &^ cfg.OmitWithFilesFlag,
	RequireAuth: true,
	PrintFlags:  true,
}

type options struct {
	full bool
	emojidl.Options
}

// emoji specific flags
var cmdFlags = options{
	Options: emojidl.Options{
		FailFast: false,
	},
}

func init() {
	CmdEmoji.Wizard = wizard
	CmdEmoji.Flag.BoolVar(&cmdFlags.FailFast, "ignore-errors", true, "ignore download errors (skip failed emojis)")
	CmdEmoji.Flag.BoolVar(&cmdFlags.full, "full", false, "fetch emojis using Edge API to get full emoji information, including usernames")
}

func run(ctx context.Context, cmd *base.Command, args []string) error {
	if err := bootstrap.AskOverwrite(cfg.Output); err != nil {
		return err
	}
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

	cmdFlags.WithFiles = cfg.WithFiles

	start := time.Now()
	r, cl := statusReporter()
	defer cl.Close()
	if cmdFlags.full {
		err = runEdge(ctx, fsa, prov, r)
	} else {
		err = runLegacy(ctx, fsa, r)
	}
	cl.Close()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	slog.InfoContext(ctx, "Emojis downloaded", "dir", cfg.Output, "took", time.Since(start).String())
	return nil
}

func statusReporter() (emojidl.StatusFunc, io.Closer) {
	pb := progressbar.NewOptions(0,
		progressbar.OptionSetDescription("Downloading emojis"),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionShowCount(),
	)
	var once sync.Once
	return func(name string, total, count int) {
		once.Do(func() {
			pb.ChangeMax(total)
		})
		pb.Add(1)
	}, pb
}

func runLegacy(ctx context.Context, fsa fsadapter.FS, cb emojidl.StatusFunc) error {
	client, err := bootstrap.Slack(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	return emojidl.DlFS(ctx, client, fsa, &cmdFlags.Options, cb)
}

func runEdge(ctx context.Context, fsa fsadapter.FS, prov auth.Provider, cb emojidl.StatusFunc) error {
	clx, err := client.NewEdge(ctx, prov)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}
	ecl, ok := clx.Edge()
	if !ok {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("edge client not available")
	}

	if err := emojidl.DlEdgeFS(ctx, ecl, fsa, &cmdFlags.Options, cb); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("application error: %s", err)
	}
	return nil
}
