package archive

import (
	"context"
	"errors"
	"strings"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/stream"
)

var CmdSearch = &base.Command{
	UsageLine: "slackdump search",
	Short:     "dump search results",
	Long:      `Searches for messages matching criteria.`,
	Commands: []*base.Command{
		cmdSearchMessages,
		cmdSearchFiles,
		cmdSearchAll,
	},
}

var cmdSearchMessages = &base.Command{
	UsageLine:   "slackdump search messages [flags] query terms",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	Run:         runSearchMsg,
	PrintFlags:  true,
}

var cmdSearchFiles = &base.Command{
	UsageLine:   "slackdump search files [flags] query terms",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	Run:         runSearchFiles,
	PrintFlags:  true,
}

var cmdSearchAll = &base.Command{
	UsageLine:   "slackdump search all [flags] query terms",
	Short:       "Searches for messages and files matching criteria. ",
	Long:        `Records search message and files results matching the given query`,
	RequireAuth: true,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	Run:         runSearchAll,
	PrintFlags:  true,
}

var fastSearch bool

func init() {
	for _, cmd := range []*base.Command{cmdSearchMessages, cmdSearchFiles, cmdSearchAll} {
		cmd.Flag.BoolVar(&fastSearch, "-no-channel-users", false, "skip channel users (approx ~2.5x faster)")
	}
}

func runSearchMsg(ctx context.Context, cmd *base.Command, args []string) error {
	ctrl, stop, err := initController(ctx, args)
	if err != nil {
		return err
	}
	defer stop()
	query := strings.Join(args, " ")
	if err := ctrl.SearchMessages(ctx, query); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func runSearchFiles(ctx context.Context, cmd *base.Command, args []string) error {
	ctrl, stop, err := initController(ctx, args)
	if err != nil {
		return err
	}
	defer stop()
	query := strings.Join(args, " ")
	if err := ctrl.SearchFiles(ctx, query); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func runSearchAll(ctx context.Context, cmd *base.Command, args []string) error {
	ctrl, stop, err := initController(ctx, args)
	if err != nil {
		return err
	}
	defer stop()
	query := strings.Join(args, " ")
	if err := ctrl.SearchAll(ctx, query); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func initController(ctx context.Context, args []string) (*control.Controller, func(), error) {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, nil, errors.New("missing query parameter")
	}

	cfg.Output = StripZipExt(cfg.Output)
	if cfg.Output == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, nil, errNoOutput
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return nil, nil, err
	}

	cd, err := chunk.CreateDir(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return nil, nil, err
	}
	defer cd.Close()

	lg := logger.FromContext(ctx)
	dl, stop := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	var sopts []stream.Option
	if fastSearch {
		sopts = append(sopts, stream.OptFastSearch())
	}

	var (
		subproc = fileproc.NewExport(fileproc.STmattermost, dl)
		stream  = sess.Stream(sopts...)
		ctrl    = control.New(cd, stream, control.WithLogger(lg), control.WithFiler(subproc))
	)
	return ctrl, stop, nil
}
