package archive

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/stream"
)

// *****
// TODO: implement database
// *****

var CmdSearch = &base.Command{
	UsageLine:   "slackdump search",
	Short:       "dump search results",
	Long:        searchMD,
	Wizard:      wizSearch,
	RequireAuth: true,
	Commands: []*base.Command{
		cmdSearchMessages,
		cmdSearchFiles,
		cmdSearchAll,
	},
}

//go:embed assets/search.md
var searchMD string

const flagMask = cfg.OmitUserCacheFlag | cfg.OmitCacheDir | cfg.OmitTimeframeFlag | cfg.OmitMemberOnlyFlag | cfg.OmitDownloadAvatarsFlag

var cmdSearchMessages = &base.Command{
	UsageLine:   "slackdump search messages [flags] <query terms>",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    flagMask | cfg.OmitRecordFilesFlag,
	Run:         runSearchFn((*control.DirController).SearchMessages),
	PrintFlags:  true,
}

var cmdSearchFiles = &base.Command{
	UsageLine:   "slackdump search files [flags]  <query terms>",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    flagMask,
	Run:         runSearchFn((*control.DirController).SearchFiles),
	PrintFlags:  true,
}

var cmdSearchAll = &base.Command{
	UsageLine:   "slackdump search all [flags]  <query terms>",
	Short:       "Searches for messages and files matching criteria. ",
	Long:        `Records search message and files results matching the given query`,
	RequireAuth: true,
	FlagMask:    flagMask,
	Run:         runSearchFn((*control.DirController).SearchAll),
	PrintFlags:  true,
}

var fastSearch bool

func init() {
	for _, cmd := range []*base.Command{cmdSearchMessages, cmdSearchFiles, cmdSearchAll} {
		cmd.Flag.BoolVar(&fastSearch, "no-channel-users", false, "skip channel users (approx ~2.5x faster)")
	}
}

var ErrNoQuery = errors.New("missing query parameter")

func runSearchFn(fn func(*control.DirController, context.Context, string) error) func(context.Context, *base.Command, []string) error {
	return func(ctx context.Context, cmd *base.Command, args []string) error {
		if len(args) == 0 {
			base.SetExitStatus(base.SInvalidParameters)
			return ErrNoQuery
		}

		cfg.Log.Info("running command", "cmd", cmd.Name())

		sess, err := bootstrap.SlackdumpSession(ctx)
		if err != nil {
			base.SetExitStatus(base.SInitializationError)
			return err
		}

		ctrl, stop, err := searchControllerv3(ctx, cfg.Output, sess, args)
		if err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		defer func() {
			if err := ctrl.Close(); err != nil {
				cfg.Log.Error("error closing controller", "err", err)
			}
		}()
		defer stop()

		query := strings.Join(args, " ")
		if err := fn(ctrl, ctx, query); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		return nil
	}
}

func searchControllerv3(ctx context.Context, dir string, sess *slackdump.Session, terms []string) (*control.DirController, func(), error) {
	noop := func() {}
	if len(terms) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, noop, errors.New("missing query parameter")
	}

	cd, err := NewDirectory(dir)
	if err != nil {
		return nil, noop, err
	}

	lg := cfg.Log

	dl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	pb := bootstrap.ProgressBar(ctx, lg, progressbar.OptionShowCount()) // progress bar

	var once sync.Once
	sopts := []stream.Option{
		stream.OptResultFn(func(sr stream.Result) error {
			lg.DebugContext(ctx, "stream", "result", sr.String())
			once.Do(func() { pb.Describe(sr.String()) })
			pb.Add(sr.Count)
			return nil
		}),
	}
	if fastSearch {
		sopts = append(sopts, stream.OptFastSearch())
	}

	ctrl := control.NewDir(
		cd,
		sess.Stream(sopts...),
		control.WithLogger(lg),
		control.WithFiler(fileproc.New(dl)),
		control.WithFlags(control.Flags{RecordFiles: cfg.RecordFiles}),
	)
	stop := func() {
		_ = pb.Finish()
		if err := cd.Close(); err != nil {
			cfg.Log.Error("error closing directory", "err", err)
		}
	}
	return ctrl, stop, nil
}

type stopFn []func() error

func (s stopFn) Close() error {
	var err error
	for i := len(s) - 1; i >= 0; i-- {
		if e := s[i](); err != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func searchControllerv31(ctx context.Context, dir string, sess *slackdump.Session, terms []string) (*control.DBController, io.Closer, error) {
	var stop stopFn
	if len(terms) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, stop, errors.New("missing query parameter")
	}

	cd, err := NewDirectory(dir)
	if err != nil {
		return nil, stop, err
	}
	db, si, err := bootstrap.Database(dir, "search")
	if err != nil {
		return nil, stop, err
	}
	stop = append(stop, db.Close)

	lg := cfg.Log

	dl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	pb := bootstrap.ProgressBar(ctx, lg, progressbar.OptionShowCount()) // progress bar
	stop = append(stop, pb.Finish)

	var once sync.Once
	sopts := []stream.Option{
		stream.OptResultFn(func(sr stream.Result) error {
			lg.DebugContext(ctx, "stream", "result", sr.String())
			once.Do(func() { pb.Describe(sr.String()) })
			pb.Add(sr.Count)
			return nil
		}),
	}
	if fastSearch {
		sopts = append(sopts, stream.OptFastSearch())
	}

	dbp, err := dbproc.New(ctx, db, si)
	if err != nil {
		return nil, stop, err
	}
	stop = append(stop, dbp.Close)

	ctrl, err := control.NewDB(
		ctx,
		sess.Stream(sopts...),
		dbp,
		control.WithLogger(lg),
		control.WithFiler(fileproc.New(dl)),
		control.WithFlags(control.Flags{RecordFiles: cfg.RecordFiles}),
	)
	if err != nil {
		return nil, stop, err
	}
	return ctrl, stop, nil
}
