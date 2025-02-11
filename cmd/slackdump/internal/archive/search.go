package archive

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
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

		cd, err := NewDirectory(cfg.Output)
		if err != nil {
			base.SetExitStatus(base.SInvalidParameters)
			return err
		}
		defer cd.Close()

		ctrl, stop, err := searchController(ctx, cd, sess, args)
		if err != nil {
			return err
		}
		defer ctrl.Close()
		defer stop()

		query := strings.Join(args, " ")
		if err := fn(ctrl, ctx, query); err != nil {
			base.SetExitStatus(base.SApplicationError)
			return err
		}
		return nil
	}
}

func searchController(ctx context.Context, cd *chunk.Directory, sess *slackdump.Session, terms []string) (*control.DirController, func(), error) {
	if len(terms) == 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, nil, errors.New("missing query parameter")
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
	return ctrl, func() { pb.Finish() }, nil
}
