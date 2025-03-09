package resume

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/archive"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
)

var CmdResume = &base.Command{
	UsageLine:   "slackdump resume [flags] <archive or directory>",
	Short:       "resume resumes archive process from the last checkpoint",
	PrintFlags:  true,
	RequireAuth: true,
}

type ResumeParams struct {
	// Refresh the list of channels from the server.  Allows
	// adding non-existing channels that appeared since the last
	// run.
	Refresh bool
	// IncludeThreads includes scanning of the threads in the archive
	// and checking if there are any new messages in them.
	IncludeThreads bool
}

var resumeFlags ResumeParams

func init() {
	CmdResume.Run = runResume
	CmdResume.Flag.BoolVar(&resumeFlags.Refresh, "refresh", false, "refresh the list of channels")
	CmdResume.Flag.BoolVar(&resumeFlags.IncludeThreads, "threads", false, "include threads (slow, and flaky business)")
}

func runResume(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("expected exactly one argument")
	}
	loc := args[0]

	flags, err := source.Type(loc)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("error determining source type: %w", err)
	}
	if !flags.Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("source type %s does not support resume", flags)
	}
	latest, err := latest(ctx, loc)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("error loading latest timestamps: %w", err)
	}
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		return fmt.Errorf("error creating slackdump session: %w", err)
	}

	wconn, _, err := bootstrap.Database(loc, cmd.Name())
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}
	defer wconn.Close()

	cf := control.Flags{
		Refresh: resumeFlags.Refresh,
	}
	ctrl, err := archive.DBController(ctx, cmd, wconn, sess, loc, cf, stream.OptInclusive(false))
	if err != nil {
		return fmt.Errorf("error creating archive controller: %w", err)
	}
	defer ctrl.Close()

	if err := ctrl.Run(ctx, latest); err != nil {
		return fmt.Errorf("error running archive controller: %w", err)
	}

	return nil
}

func latest(ctx context.Context, srcpath string) (*structures.EntityList, error) {
	src, err := source.Load(ctx, srcpath)
	if err != nil {
		return nil, fmt.Errorf("error loading source: %w", err)
	}
	defer src.Close()

	latest, err := src.Latest(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading latest timestamps: %w", err)
	}

	if cfg.Verbose {
		strlatest(latest)
	}

	ei := make([]structures.EntityItem, 0, len(latest))
	for sl, ts := range latest {
		if sl.IsThread() && !resumeFlags.IncludeThreads {
			continue
		}
		item := structures.EntityItem{
			Id:      sl.String(),
			Oldest:  ts,
			Latest:  time.Time(cfg.Latest),
			Include: true,
		}
		ei = append(ei, item)
		debugprint(fmt.Sprintf("%s: %d->%d", item.Id, ts.UTC().UnixMicro(), item.Oldest.UnixMicro()))
	}
	el := structures.NewEntityListFromItems(ei...)

	return el, nil
}

func debugprint(a ...any) {
	if cfg.Verbose {
		fmt.Println(a...)
	}
}

func strlatest(l map[structures.SlackLink]time.Time) string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(tw, "Group ID\tLatest")
	for gid, ts := range l {
		fmt.Fprintf(tw, "%s\t%s\n", gid, ts.Format("2006-01-02 15:04:05 MST"))
	}
	tw.Flush()
	return buf.String()
}
