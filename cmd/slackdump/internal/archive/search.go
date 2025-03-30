package archive

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"sync"

	fileproc2 "github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/stream"
)

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

const flagMask = cfg.OmitUserCacheFlag | cfg.OmitCacheDir | cfg.OmitTimeframeFlag | cfg.OmitCustomUserFlags | cfg.OmitDownloadAvatarsFlag

var cmdSearchMessages = &base.Command{
	UsageLine:   "slackdump search messages [flags] <query terms>",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    flagMask | cfg.OmitRecordFilesFlag,
	Run:         runSearchMsg,
	PrintFlags:  true,
}

var cmdSearchFiles = &base.Command{
	UsageLine:   "slackdump search files [flags]  <query terms>",
	Short:       "records search results matching the given query",
	Long:        `Searches for messages matching criteria.`,
	RequireAuth: true,
	FlagMask:    flagMask,
	Run:         runSearchFiles,
	PrintFlags:  true,
}

var cmdSearchAll = &base.Command{
	UsageLine:   "slackdump search all [flags]  <query terms>",
	Short:       "Searches for messages and files matching criteria. ",
	Long:        `Records search message and files results matching the given query`,
	RequireAuth: true,
	FlagMask:    flagMask,
	Run:         runSearchAll,
	PrintFlags:  true,
}

var fastSearch bool

func init() {
	for _, cmd := range []*base.Command{cmdSearchMessages, cmdSearchFiles, cmdSearchAll} {
		cmd.Flag.BoolVar(&fastSearch, "no-channel-users", false, "skip channel users (approx ~2.5x faster)")
	}
}

func runSearchMsg(ctx context.Context, cmd *base.Command, args []string) error {
	return runSearch(ctx, cmd, args, control.SMessages)
}

func runSearchFiles(ctx context.Context, cmd *base.Command, args []string) error {
	return runSearch(ctx, cmd, args, control.SFiles)
}

func runSearchAll(ctx context.Context, cmd *base.Command, args []string) error {
	return runSearch(ctx, cmd, args, control.SMessages|control.SFiles)
}

var ErrNoQuery = errors.New("missing query parameter")

func runSearch(ctx context.Context, cmd *base.Command, args []string, typ control.SearchType) error {
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

	ctrl, stop, err := searchControllerv31(ctx, cfg.Output, sess, args)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	defer stop.Stop()
	defer func() {
		if err := ctrl.Close(); err != nil {
			cfg.Log.Error("error closing controller", "err", err)
		}
	}()

	query := strings.Join(args, " ")
	if err := ctrl.Search(ctx, query, typ); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

type stopFn []func() error

func (s stopFn) Stop() error {
	var err error
	for i := len(s) - 1; i >= 0; i-- {
		if e := s[i](); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func searchControllerv31(ctx context.Context, dir string, sess *slackdump.Session, terms []string) (*control.Controller, stopFn, error) {
	var stop stopFn
	if len(terms) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, stop, errors.New("missing query parameter")
	}

	cd, err := NewDirectory(dir)
	if err != nil {
		return nil, stop, err
	}
	stop = append(stop, cd.Close)
	db, si, err := bootstrap.Database(cd.Name(), "search")
	if err != nil {
		return nil, stop, err
	}
	stop = append(stop, db.Close)

	lg := cfg.Log

	dl := fileproc2.NewDownloader(
		ctx,
		cfg.WithFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	erc, err := dbase.New(ctx, db, si)
	if err != nil {
		return nil, stop, err
	}

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

	ctrl, err := control.New(
		ctx,
		sess.Stream(sopts...),
		erc,
		control.WithLogger(lg),
		control.WithFiler(fileproc2.New(dl)),
		control.WithFlags(control.Flags{RecordFiles: cfg.RecordFiles}),
	)
	if err != nil {
		return nil, stop, err
	}
	return ctrl, stop, nil
}
