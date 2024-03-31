package record

import (
	"context"
	"errors"
	"strings"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/logger"
)

var CmdSearch = &base.Command{
	UsageLine:   "slackdump search [flags] query terms",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	Run:         runSearch,
	PrintFlags:  true,
}

func runSearch(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("missing query parameter")
	}
	query := strings.Join(args, " ")

	cfg.Output = stripZipExt(cfg.Output)
	if cfg.Output == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errNoOutput
	}

	sess, err := cfg.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	cd, err := chunk.CreateDir(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
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
	defer stop()
	var (
		subproc = fileproc.NewExport(fileproc.STmattermost, dl)
		stream  = sess.Stream()
		ctrl    = control.New(cd, stream, control.WithLogger(lg), control.WithFiler(subproc))
	)
	if err := ctrl.Search(ctx, query); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}
