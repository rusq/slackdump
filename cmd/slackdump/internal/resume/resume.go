package resume

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/archive"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
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
	CmdResume.Flag.BoolVar(&resumeFlags.IncludeThreads, "threads", false, "include threads")
}

func runResume(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("expected exactly one argument")
	}
	archive := args[0]

	flags, err := source.Type(archive)
	if err != nil {
		return fmt.Errorf("error determining source type: %w", err)
	}

	src, err := source.Load(ctx, archive)
	if err != nil {
		return fmt.Errorf("error loading source: %w", err)
	}
	defer src.Close()

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		return fmt.Errorf("error creating slackdump session: %w", err)
	}

	if err := Resume(ctx, sess, src, flags, resumeFlags); err != nil {
		return fmt.Errorf("error resuming archive: %w", err)
	}
	return nil
}

func Resume(ctx context.Context, sess *slackdump.Session, src source.Sourcer, flags source.Flags, p ResumeParams) error {
	lg := cfg.Log.With("source", src.Name(), "flags", src.Type())
	lg.Info("resuming archive")
	channels, err := src.Channels(ctx)
	if err != nil {
		return fmt.Errorf("error loading channels: %w", err)
	}
	lg.Info("channels loaded", "count", len(channels))

	// start catching up on existing channels
	if p.Refresh {
		lg.Info("fetching latest channels")
		// start fetching channels from the server
	}

	lg.Info("scanning messages")

	latest, err := src.Latest(ctx)
	if err != nil {
		return fmt.Errorf("error loading latest timestamps: %w", err)
	}

	// by this point we have all the channels and maybe threads along with their
	// respective latest timestamps.
	debugprint(strlatest(latest))
	// remove all threads from the list if they are disabled
	el := make([]structures.EntityItem, 0, len(latest))
	for sl, ts := range latest {
		if sl.IsThread() && !p.IncludeThreads {
			continue
		}
		item := structures.EntityItem{
			Id:      sl.String(),
			Oldest:  ts,
			Latest:  time.Time(cfg.Latest),
			Include: true,
		}
		el = append(el, item)
		debugprint(fmt.Sprintf("%s: %d->%d", item.Id, ts.UnixMicro(), item.Oldest.UnixMicro()))
	}
	list := structures.NewEntityListFromItems(el...)

	cd, err := archive.NewDirectory(cfg.Output)
	if err != nil {
		return fmt.Errorf("error creating archive directory: %w", err)
	}
	defer cd.Close()

	ctrl, err := archive.ArchiveController(ctx, cd, sess, stream.OptInclusive(false))
	if err != nil {
		return fmt.Errorf("error creating archive controller: %w", err)
	}
	defer ctrl.Close()
	if err := ctrl.Run(ctx, list); err != nil {
		return fmt.Errorf("error running archive controller: %w", err)
	}

	return nil
}

func debugprint(a ...any) {
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
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
