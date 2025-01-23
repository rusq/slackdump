package resume

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/source"
)

var CmdResume = &base.Command{
	UsageLine:   "slackdump resume [flags] <archive or directory>",
	Short:       "resume resumes archive process from the last checkpoint",
	FlagMask:    cfg.OmitAll &^ (cfg.OmitAuthFlags | cfg.OmitCacheDir | cfg.OmitDownloadFlag | cfg.OmitDownloadAvatarsFlag),
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
	lg := cfg.Log.With("source", src.Name())
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

	l, err := src.Latest(ctx)
	if err != nil {
		return fmt.Errorf("error loading latest timestamps: %w", err)
	}

	fmt.Println(l)

	// by this point we have all the channels and maybe threads along with their
	// respective latest timestamps.

	return nil
}
